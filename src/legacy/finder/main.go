// This app looks all over the filesystem and the database to figure out if an
// issue exists somewhere in the process.  This is to help find issues we
// expected to see in production but haven't (in case they got "stuck" in some
// step) or where we have a dupe but aren't sure where all versions exist.

package main

import (
	"config"
	"db"
	"fileutil"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"wordutils"

	"github.com/jessevdk/go-flags"
)

// Conf stores the configuration data read from the legacy Python settings
var Conf *config.Config
var issueSearchKeys []*issueSearchKey

// Command-line options
var opts struct {
	ConfigFile string   `short:"c" long:"config" description:"path to P2C config file" required:"true"`
	Siteroot   string   `long:"siteroot" description:"URL to the live host" required:"true"`
	CachePath  string   `long:"cache-path" description:"Path to cache downloaded JSON files" required:"true"`
	IssueList  string   `long:"issue-list" description:"path to file containing list of newline-separated issue keys"`
	IssueKeys  []string `long:"issue-key" description:"single issue key to process, e.g., 'sn12345678/1905123101'"`
}

var p *flags.Parser

// wrap is a helper to wrap a usage message at 80 characters and print a
// newline afterward
func wrap(msg string) {
	fmt.Fprint(os.Stderr, wordutils.Wrap(msg, 80))
	fmt.Fprintln(os.Stderr)
}

func usageFail(format string, args ...interface{}) {
	wrap(fmt.Sprintf(format, args...))
	fmt.Fprintln(os.Stderr)
	p.WriteHelp(os.Stderr)
	fmt.Fprintln(os.Stderr)
	wrap("At least one of --issue-list or --issue-key must be specified.  " +
		"If both are specified, --issue-key will be ignored.  Note that " +
		"--issue-key may be specified multiple times.")
	fmt.Fprintln(os.Stderr)
	wrap("Issue keys MUST be formatted as LCCN[/YYYY][MM][DD][EE].  The full " +
		"LCCN is mandatory, while the rest of the key's parts can be added to " +
		"refine the search.")
	fmt.Fprintln(os.Stderr)
	wrap("--siteroot must point to the live site, for downloading batch and " +
		"issue information so the search knows if an issue is live, and if so, " +
		"in what batch it was ingested.")
	os.Exit(1)
}

func getConf() {
	p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	p.Usage = "[OPTIONS]"
	var _, err = p.Parse()

	if err != nil {
		usageFail("Error: %s", err)
	}

	if len(opts.IssueKeys) == 0 && opts.IssueList == "" {
		usageFail("Error: You must specify one or more issue keys via --issue-keys or --issue-list")
	}

	Conf, err = config.Parse(opts.ConfigFile)
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	err = db.Connect(Conf.DatabaseConnect)
	if err != nil {
		log.Fatalf("Error trying to connect to database: %s", err)
	}

	// If we have an issue list, read it into opts.IssueKeys
	if opts.IssueList != "" {
		var contents, err = ioutil.ReadFile(opts.IssueList)
		if err != nil {
			usageFail("Unable to open issue list file %#v: %s", opts.IssueList, err)
		}
		opts.IssueKeys = strings.Split(string(contents), "\n")
	}

	// If we have a batch URL, we must have a valid cache path
	if opts.Siteroot != "" {
		if !fileutil.IsDir(opts.CachePath) {
			usageFail("ERROR: --cache-path %#v is not a valid directory", opts.CachePath)
		}
	}

	// Verify that each issue key at least *looks* legit before burning time
	// searching stuff
	for _, ik := range opts.IssueKeys {
		if ik == "" {
			continue
		}

		var searchKey, err = parseSearchKey(ik)
		if err != nil {
			usageFail("Invalid issue search key %#v: %s", ik, err)
		}
		issueSearchKeys = append(issueSearchKeys, searchKey)
	}

	if len(issueSearchKeys) == 0 {
		usageFail("No valid issue keys were found (did you use a blank issue key?)")
	}
}

func main() {
	getConf()
	cacheDBTitles()

	var err = cacheLiveBatchedIssues(opts.Siteroot, opts.CachePath)
	if err != nil {
		log.Fatalf("Error trying to cache live batched issues: %s", err)
	}

	cacheAllFilesystemIssues()
	for _, k := range issueSearchKeys {
		log.Printf("DEBUG: Looking up by issue key %#v", k.String())
		for _, ik := range k.issueKeys() {
			var issues = issueLookup[ik]
			for _, issue := range issues {
				var fsPaths = filesystemIssueLocations[ik]
				var webURLs = webIssueLocations[ik]

				fmt.Printf("- Found issue %#v:\n", ik)
				for _, batch := range issue.Batches {
					var locs []string
					if liveBatches[batch.Fullname()] != nil {
						locs = append(locs, "live site")
					}
					if filesystemBatches[batch.Fullname()] != nil {
						locs = append(locs, "filesystem")
					}

					fmt.Printf("  - In batch %#v (%s)\n", batch.Fullname(), strings.Join(locs, ", "))
				}
				for _, fsPath := range fsPaths {
					fmt.Printf("  - Filesystem: %#v\n", fsPath)
				}
				for _, webURL := range webURLs {
					fmt.Printf("  - Web: %#v\n", webURL)
				}
			}
		}
	}
}
