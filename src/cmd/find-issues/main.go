// This app reads the finder cache to report where in the process an issue (or
// group of issues) was last seen.  This is to help find issues we expected to
// see in production but haven't (in case they got "stuck" in some step) or
// where we have a dupe but aren't sure where all versions exist.

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
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
var titles models.TitleList

func getOpts() {
	var c = cli.New(&opts)
	c.AppendUsage("At least one of --all, --issue-list, or --issue-key must be " +
		"specified.  --all takes precedence over --issue-list, which takes precedence " +
		"over --issue-key.  Note that --issue-key may be specified multiple times.")
	c.AppendUsage("Issue keys MUST be formatted as LCCN[/YYYY][MM][DD][EE].  The full " +
		"LCCN is mandatory, while the rest of the key's parts can be added to " +
		"refine the search.")
	conf = c.GetConf()

	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}

	if len(opts.IssueKeys) == 0 && opts.IssueList == "" && !opts.All {
		c.UsageFail("Error: You must specify one or more issue keys via --all, --issue-key, or --issue-list")
	}

	// If we have an issue list, read it into opts.IssueKeys
	if opts.IssueList != "" {
		var contents, err = os.ReadFile(opts.IssueList)
		if err != nil {
			c.UsageFail("Unable to open issue list file %#v: %s", opts.IssueList, err)
		}
		opts.IssueKeys = strings.Split(string(contents), "\n")
	}

	titles, err = models.Titles()
	if err != nil {
		c.UsageFail("Unable to read titles from the database: %s", err)
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

		// Find the title in the database via our generic title finder and make
		// sure we really have an LCCN in the search key
		var t = titles.Find(searchKey.LCCN)
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

	var list schema.IssueList
	for _, k := range issueSearchKeys {
		var found = scanner.LookupIssues(k)
		if len(found) > 0 {
			list = append(list, found...)
		} else {
			log.Printf("Error: issue key %q has no matches", k)
		}
	}
	reportIssues(list)
}

type locData struct {
	Location string
	Batch    string
	Errors   []string
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

	var locs = make(map[string][]*locData)
	for _, issue := range newList {
		var key = issue.Key()
		var loc = &locData{Location: issue.Location}
		if issue.Batch != nil {
			loc.Batch = issue.Batch.Fullname()
		}

		for _, e := range issue.Errors.All() {
			var word = "ERROR"
			if e.Warning() {
				word = "WARNING"
			}
			loc.Errors = append(loc.Errors, word+": "+e.Error())
		}

		locs[key] = append(locs[key], loc)
	}

	var data, err = json.MarshalIndent(locs, "", "\t")
	if err != nil {
		log.Fatalf("Error marshaling location report: %s", err)
	}
	fmt.Println(string(data))
}
