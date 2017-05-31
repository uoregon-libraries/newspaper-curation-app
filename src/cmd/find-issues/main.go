// This app reads the finder cache to report where in the process an issue (or
// group of issues) was last seen.  This is to help find issues we expected to
// see in production but haven't (in case they got "stuck" in some step) or
// where we have a dupe but aren't sure where all versions exist.

package main

import (
	"config"
	"db"
	"fileutil"
	"fmt"
	"io/ioutil"
	"issuefinder"
	"issuesearch"
	"log"
	"os"
	"strings"
	"wordutils"

	"github.com/jessevdk/go-flags"
)

var issueSearchKeys []*issuesearch.Key

// Conf stores the configuration data read from the legacy Python settings
var Conf *config.Config

// Command-line options
var opts struct {
	ConfigFile string `short:"c" long:"config" description:"path to P2C config file" required:"true"`
	CacheFile string   `long:"cache-file" description:"Path to the finder cache" required:"true"`
	IssueList string   `long:"issue-list" description:"path to file containing list of newline-separated issue keys"`
	IssueKeys []string `long:"issue-key" description:"single issue key to process, e.g., 'sn12345678/1905123101'"`
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
	os.Exit(1)
}

func getOpts() {
	p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	p.Usage = "[OPTIONS]"
	var _, err = p.Parse()

	if err != nil {
		usageFail("Error: %s", err)
	}

	Conf, err = config.Parse(opts.ConfigFile)
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	err = db.Connect(Conf.DatabaseConnect)
	if err != nil {
		log.Fatalf("Error trying to connect to database: %s", err)
	}

	if len(opts.IssueKeys) == 0 && opts.IssueList == "" {
		usageFail("Error: You must specify one or more issue keys via --issue-keys or --issue-list")
	}

	// If we have an issue list, read it into opts.IssueKeys
	if opts.IssueList != "" {
		var contents, err = ioutil.ReadFile(opts.IssueList)
		if err != nil {
			usageFail("Unable to open issue list file %#v: %s", opts.IssueList, err)
		}
		opts.IssueKeys = strings.Split(string(contents), "\n")
	}

	if !fileutil.IsFile(opts.CacheFile) {
		usageFail("ERROR: --cache-file %#v is not a valid file", opts.CacheFile)
	}

	// Verify that each issue key at least *looks* legit before burning time
	// searching stuff
	for _, ik := range opts.IssueKeys {
		if ik == "" {
			continue
		}

		var searchKey, err = issuesearch.ParseSearchKey(ik)
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
	getOpts()
	var finder, err = issuefinder.Deserialize(opts.CacheFile)
	if err != nil {
		log.Fatalf("Unable to deserialize the cache file %#v: %s", opts.CacheFile, err)
	}

	var lookup = issuesearch.NewLookup()
	lookup.Populate(finder.Issues)
	finder.Errors.Index()

	var lastKey = ""
	for _, k := range issueSearchKeys {
		for _, issue := range lookup.Issues(k) {
			var currKey = issue.Key()
			if currKey != lastKey {
				fmt.Printf("%#v:\n", currKey)
				lastKey = currKey
			}
			fmt.Printf("  - %#v\n", issue.Location)
			if issue.Batch != nil {
				fmt.Printf("    - Batch: %s\n", issue.Batch.Fullname())
			}

			var errors = finder.Errors.IssueErrors[issue]
			for _, e := range errors {
				fmt.Printf("    - ERROR: (%#v) %s\n", e.Location, e.Error)
			}
		}
	}
}
