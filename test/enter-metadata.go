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
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

var conf *config.Config

var opts cli.BaseOptions
var l = logger.New(logger.Debug, false)

func getOpts() {
	var c = cli.New(&opts)
	conf = c.GetConf()

	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		l.Fatalf("Error trying to connect to database: %s", err)
	}
}

func main() {
	getOpts()

	// We grab all issues here so the debug output is helpful
	var issues, err = models.Issues().Fetch()
	if err != nil {
		l.Fatalf("Unable to find in-process issues to enter dummy metadata: %s", err)
	}
	var process []*models.Issue
	for _, i := range issues {
		if i.WorkflowStep == schema.WSReadyForMetadataEntry {
			process = append(process, i)
			l.Debugf("Queueing issue %s", i.Key())
		} else {
			l.Debugf("Skipping issue %s: workflow step is %s", i.Key(), i.WorkflowStep)
		}
	}
	if len(process) == 0 {
		l.Infof("No issues were found that need metadata entry; exiting")
		os.Exit(0)
	}

	var u = models.FindActiveUserWithLogin("admin")
	if u == nil {
		l.Fatalf("Cannot enter dummy metadata: no user")
	}

	for _, i := range process {
		l.Infof("Entering metadata for %s", i.Key())
		curate(u, i)
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
	for _ = range pages {
		i.PageLabels = append(i.PageLabels, "0")
	}

	i.QueueForMetadataReview(u.ID)
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
