// This app reads the finder cache to report where in the process an issue (or
// group of issues) was last seen.  This is to help find issues we expected to
// see in production but haven't (in case they got "stuck" in some step) or
// where we have a dupe but aren't sure where all versions exist.

package main

import (
	"config"
	"db"
	"fmt"
	"io/ioutil"
	"issuefinder"
	"issuesearch"
	"issuewatcher"
	"os"
	"schema"
	"strings"

	"github.com/jessevdk/go-flags"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/wordutils"
)

var issueSearchKeys []*issuesearch.Key

var conf *config.Config

// Command-line options
var opts struct {
	ConfigFile string   `short:"c" long:"config" description:"path to Black Mamba config file" required:"true"`
	NotLive    bool     `long:"not-live" description:"don't report live issues"`
	All        bool     `long:"all" description:"report all issues (unless --not-live is present)"`
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
	wrap("At least one of --all, --issue-list, or --issue-key must be specified.  " +
		"--all takes precedence over --issue-list, which takes precedence over " +
		"--issue-key.  Note that --issue-key may be specified multiple times.")
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

	conf, err = config.Parse(opts.ConfigFile)
	if err != nil {
		logger.Fatalf("Config error: %s", err)
	}

	err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}

	if len(opts.IssueKeys) == 0 && opts.IssueList == "" && !opts.All {
		usageFail("Error: You must specify one or more issue keys via --all, --issue-key, or --issue-list")
	}

	// If we have an issue list, read it into opts.IssueKeys
	if opts.IssueList != "" {
		var contents, err = ioutil.ReadFile(opts.IssueList)
		if err != nil {
			usageFail("Unable to open issue list file %#v: %s", opts.IssueList, err)
		}
		opts.IssueKeys = strings.Split(string(contents), "\n")
	}

	// Verify that each issue key at least *looks* legit before burning time
	// searching stuff
	for _, ik := range opts.IssueKeys {
		if ik == "" {
			continue
		}

		// Just to be nice, let's strip dashes so it's easier to paste in dates
		ik = strings.Replace(ik, "-", "", -1)
		var searchKey, err = issuesearch.ParseSearchKey(ik)
		if err != nil {
			usageFail("Invalid issue search key %#v: %s", ik, err)
		}
		issueSearchKeys = append(issueSearchKeys, searchKey)
	}

	if opts.All == true {
		return
	}

	if len(issueSearchKeys) == 0 {
		usageFail("No valid issue keys were found (did you use a blank issue key?)")
	}
}

type errorFn func(*schema.Issue) []*issuefinder.Error

func main() {
	getOpts()
	var watcher = issuewatcher.New(conf)
	var err = watcher.Deserialize()
	if err != nil {
		logger.Fatalf("Unable to deserialize the watcher: %s", err)
	}

	if opts.All {
		reportIssues(watcher.IssueFinder().Issues, watcher.IssueErrors)
		return
	}

	for _, k := range issueSearchKeys {
		reportIssues(watcher.LookupIssues(k), watcher.IssueErrors)
	}
}

func reportIssues(issueList schema.IssueList, errfn errorFn) {
	var newList = issueList
	if opts.NotLive {
		newList = make(schema.IssueList, 0)
		for _, issue := range issueList {
			if !issue.IsLive() {
				newList = append(newList, issue)
			}
		}
	}

	var lastKey = ""
	for _, issue := range newList {
		var currKey = issue.Key()
		if currKey != lastKey {
			fmt.Printf("%#v:\n", currKey)
			lastKey = currKey
		}
		fmt.Printf("  - %#v\n", issue.Location)
		if issue.Batch != nil {
			fmt.Printf("    - Batch: %s\n", issue.Batch.Fullname())
		}

		var errors = errfn(issue)
		for _, e := range errors {
			fmt.Printf("    - ERROR: (%#v) %s\n", e.Location, e.Error)
		}
	}
}
