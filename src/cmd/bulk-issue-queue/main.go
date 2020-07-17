package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
	"github.com/uoregon-libraries/newspaper-curation-app/src/uploads"
)

// Command-line options
type _opts struct {
	cli.BaseOptions
	Type string `long:"type" description:"Type of issues to queue: 'scan' or 'borndigital'" required:"true"`
	Key  string `long:"key" description:"Issue key for which issues are to be considered"`
}

var opts _opts
var scanner *issuewatcher.Scanner
var conf *config.Config

func getOpts() {
	var c = cli.New(&opts)
	c.AppendUsage(`Looks at all issues of the given type ("scan" or "borndigital") ` +
		`and for the given key (LCCN with optional date components; e.g., ` +
		`"sn12345678", "sn12345678/18900101", etc.) and attempts to push them ` +
		"into the NCA workflow if they are error-free.")
	c.AppendUsage("If --key is omitted, a list of titles will be displayed instead of taking action.")

	conf = c.GetConf()

	var err = dbi.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}

	scanner = issuewatcher.NewScanner(conf).DisableDB().DisableWeb()
	switch opts.Type {
	case "scan":
		scanner.DisableSFTPUpload()
	case "borndigital":
		scanner.DisableScannedUpload()
	default:
		c.UsageFail("%q is not a valid issue type", opts.Type)
	}
}

func getIssues() schema.IssueList {
	var err error

	var iKey *schema.Key
	if opts.Key != "" {
		iKey, err = schema.ParseSearchKey(opts.Key)
		if err != nil {
			logger.Fatalf("Invalid key %q: %s", opts.Key, err)
		}
	}

	err = scanner.Scan()
	if err != nil {
		logger.Fatalf("unable to scan filesystem: %s", err)
	}

	var lccnIssues schema.IssueList
	if iKey != nil {
		lccnIssues = scanner.LookupIssues(iKey)
	}

	return lccnIssues
}

func reportTitles() {
	var allIssues = scanner.Finder.Issues
	var lccnsSeen = make(map[*schema.Title]int)
	for _, issue := range allIssues {
		lccnsSeen[issue.Title]++
	}
	for t, n := range lccnsSeen {
		fmt.Printf("%s (%s), %d issue(s) found\n", t.LCCN, t.Name, n)
	}
}

func main() {
	getOpts()
	var issues = getIssues()
	if len(issues) == 0 {
		fmt.Println("Valid Titles:")
		fmt.Println("------------")
		reportTitles()
		os.Exit(0)
	}

	logger.Infof("Reading issue data - this can take a long time if the web issue cache hasn't been built previously")
	var globalScanner = issuewatcher.NewScanner(conf)
	var err = globalScanner.Scan()
	if err != nil {
		logger.Fatalf("Error trying to scan issues: %s", err)
	}
	logger.Infof("Done reading issue data")

	for _, issue := range issues {
		var i = uploads.New(issue, globalScanner, conf)
		logger.Infof("Looking at issue %q", i.Key())
		i.ValidateAll()
		if len(i.Errors) != 0 {
			var errorList []string
			for _, e := range i.Errors {
				errorList = append(errorList, e.Message())
			}
			logger.Warnf("Skipping %q: %s", i.Key(), strings.Join(errorList, "; "))
			continue
		}

		var err = i.Queue()
		if err != nil {
			logger.Warnf("Skipping %q: %s", i.Key(), err)
		}
	}
}
