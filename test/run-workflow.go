//go:build ignore

package main

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

var conf *config.Config

type _opts struct {
	cli.BaseOptions
	Operation string `long:"operation" description:"Type of operation to perform: 'curate' or 'review'" required:"true"`
}

var opts _opts
var l = logger.New(logger.Debug, false)
var process func(*models.User, *models.Issue)
var ws schema.WorkflowStep

func getOpts() {
	var c = cli.New(&opts)
	conf = c.GetConf()

	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		l.Fatalf("Error trying to connect to database: %s", err)
	}

	switch opts.Operation {
	case "curate":
		l.Infof("Will curate all issues needing metadata entry")
		process = curate
		ws = schema.WSReadyForMetadataEntry
	case "review":
		l.Infof("Will approve all issues which have been queued for review")
		process = review
		ws = schema.WSAwaitingMetadataReview
	default:
		c.UsageFail("%q is not a valid operation", opts.Operation)
	}
}

func main() {
	getOpts()

	// We grab all issues here so the debug output is helpful
	var allIssues, err = models.Issues().Fetch()
	if err != nil {
		l.Fatalf("Unable to find in-process issues in the database: %s", err)
	}
	var toProcess []*models.Issue
	for _, i := range allIssues {
		if i.WorkflowStep == ws {
			toProcess = append(toProcess, i)
			l.Debugf("Queueing issue %s", i.Key())
		} else {
			l.Debugf("Skipping issue %s: workflow step is %q (we want %q)", i.Key(), i.WorkflowStep, ws)
		}
	}
	if len(toProcess) == 0 {
		l.Infof("No issues were found in %q; exiting", ws)
		os.Exit(0)
	}

	var u = models.FindActiveUserWithLogin("admin")
	if u == nil {
		l.Fatalf("Cannot enter dummy metadata: no user")
	}

	for _, i := range toProcess {
		l.Infof("Processing %s", i.Key())
		process(u, i)
	}
}

func curate(u *models.User, i *models.Issue) {
	i.Claim(u.ID)
	i.Issue = "0"
	i.Volume = "0"
	i.EditionLabel = ""
	i.DateAsLabeled = i.Date
	i.Edition = 1
	i.PageLabels = nil

	var pages, err = getFiles(i.Location, ".pdf")
	if err != nil {
		l.Fatalf("Cannot read pages: %s", err)
	}
	for range pages {
		i.PageLabels = append(i.PageLabels, "0")
	}

	err = i.QueueForMetadataReview(u.ID)
	if err != nil {
		l.Fatalf("Unable to queue issue %s for review: %s", i.Key(), err)
	}
}

func review(u *models.User, i *models.Issue) {
	i.Claim(u.ID)
	var err = i.ApproveMetadata(u.ID)
	if err != nil {
		l.Fatalf("Unable to approve metadata for issue %s: %s", i.Key(), err)
	}
	err = jobs.QueueFinalizeIssue(i)
	if err != nil {
		l.Fatalf("Unable to queue issue finalization job for issue %s: %s", i.Key(), err)
	}
}

func getFiles(dir string, exts ...string) ([]string, error) {
	var fileList, err = fileutil.FindIf(dir, func(i os.FileInfo) bool {
		for _, ext := range exts {
			if filepath.Ext(i.Name()) == ext {
				return true
			}
		}
		return false
	})

	sort.Strings(fileList)
	return fileList, err
}
