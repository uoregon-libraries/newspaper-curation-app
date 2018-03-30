// This app reads the finder cache to report where in the process an issue (or
// group of issues) was last seen.  This is to help find issues we expected to
// see in production but haven't (in case they got "stuck" in some step) or
// where we have a dupe but aren't sure where all versions exist.

package main

import (
	"cli"
	"config"
	"db"
	"fmt"
	"io/ioutil"
	"issuewatcher"
	"schema"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
)

var issueSearchKeys []*schema.Key

var conf *config.Config

// Command-line options
type _opts struct {
	cli.BaseOptions
	NotLive   bool     `long:"not-live" description:"don't report live issues"`
	All       bool     `long:"all" description:"report all issues (unless --not-live is present)"`
	IssueList string   `long:"issue-list" description:"path to file containing list of newline-separated issue keys"`
	IssueKeys []string `long:"issue-key" description:"single issue key to process, e.g., 'sn12345678/1905123101'"`
}

var opts _opts

func getOpts() {
	var c = cli.New(&opts)
	c.AppendUsage("At least one of --all, --issue-list, or --issue-key must be " +
		"specified.  --all takes precedence over --issue-list, which takes precedence " +
		"over --issue-key.  Note that --issue-key may be specified multiple times.")
	c.AppendUsage("Issue keys MUST be formatted as LCCN[/YYYY][MM][DD][EE].  The full " +
		"LCCN is mandatory, while the rest of the key's parts can be added to " +
		"refine the search.")
	conf = c.GetConf()

	var err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}

	if len(opts.IssueKeys) == 0 && opts.IssueList == "" && !opts.All {
		c.UsageFail("Error: You must specify one or more issue keys via --all, --issue-key, or --issue-list")
	}

	// If we have an issue list, read it into opts.IssueKeys
	if opts.IssueList != "" {
		var contents, err = ioutil.ReadFile(opts.IssueList)
		if err != nil {
			c.UsageFail("Unable to open issue list file %#v: %s", opts.IssueList, err)
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
		var searchKey, err = schema.ParseSearchKey(ik)
		if err != nil {
			c.UsageFail("Invalid issue search key %#v: %s", ik, err)
		}

		// See if a title's directory was given and convert to LCCN if so
		var t, _ = db.FindTitleByDirectory(searchKey.LCCN)
		if t != nil {
			searchKey.LCCN = t.LCCN
		}

		issueSearchKeys = append(issueSearchKeys, searchKey)
	}

	if opts.All == true {
		return
	}

	if len(issueSearchKeys) == 0 {
		c.UsageFail("No valid issue keys were found (did you use a blank issue key?)")
	}
}

func main() {
	getOpts()
	var scanner = issuewatcher.NewScanner(conf)
	if !fileutil.Exists(scanner.CacheFile()) {
		logger.Fatalf("Unable to deserialize the scanner: %s cannot be read", scanner.CacheFile())
	}

	var err = scanner.Deserialize()
	if err != nil {
		logger.Fatalf("Unable to deserialize the scanner: %s", err)
	}

	if opts.All {
		reportIssues(scanner.Finder.Issues)
		return
	}

	for _, k := range issueSearchKeys {
		reportIssues(scanner.LookupIssues(k))
	}
}

func reportIssues(issueList schema.IssueList) {
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

		for _, e := range issue.Errors {
			fmt.Printf("    - ERROR: %s\n", e)
		}
	}
}
