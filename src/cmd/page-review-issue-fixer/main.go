package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// Command-line options
type _opts struct {
	cli.BaseOptions
	Keys    []string `long:"key" description:"Issue(s) to remove"`
	KeyFile string   `long:"key-file" description:"File with one key per line"`
}

var opts _opts
var conf *config.Config

// this is dumb but it makes multi-line usage into a single line
func normalize(s string) string {
	var r = regexp.MustCompile(`\s\s+`)
	return r.ReplaceAllString(s, " ")
}

func getConfig() {
	var c = cli.New(&opts)
	c.AppendUsage(normalize(
		`Removes issues from the page-review location, moving them to the error
		location and cleaning up the database as necessary. "--key" may be
		specified multiple times, but each key must be full (LCCN + slash +
		date-edition, e.g., sn12345678/2020010101) to avoid accidentally deleting
		anything.`))

	conf = c.GetConf()
	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}

	if len(opts.Keys) == 0 && opts.KeyFile == "" {
		c.UsageFail("Error: one or more keys, or a key file, must be specified")
	}
	if len(opts.Keys) > 0 && opts.KeyFile != "" {
		c.UsageFail("Error: you must specify keys or a key file, not both")
	}

	if opts.KeyFile != "" {
		var data, err = os.ReadFile(opts.KeyFile)
		if err != nil {
			logger.Fatalf("Unable to read key file %q: %s", opts.KeyFile, err)
		}

		for _, k := range strings.Split(string(data), "\n") {
			var key = strings.TrimSpace(k)
			if key != "" {
				opts.Keys = append(opts.Keys, key)
			}
		}
	}
}

func main() {
	getConfig()

	var issues, err = getIssues(opts.Keys)

	if err != nil {
		logger.Fatalf("There were one or more errors retrieving issues: %s", err)
	}

	for _, i := range issues {
		err = removeIssue(i)
		if err != nil {
			logger.Errorf("Unable to remove %q: %s", i.Key(), err)
		} else {
			logger.Infof("Removing issue %q from disk and database", i.Key())
		}
	}
}

func getIssues(keys []string) ([]*models.Issue, error) {
	var errors []string
	var issues []*models.Issue

	for _, k := range keys {
		var i, err = models.FindIssueByKey(k)
		if err != nil {
			errors = append(errors, fmt.Sprintf("unable to retrieve issue: %s", err))
			continue
		}
		if i == nil {
			errors = append(errors, fmt.Sprintf("unable to retrieve issue: issue key %q not found", k))
			continue
		}
		if i.WorkflowStep != schema.WSAwaitingPageReview {
			errors = append(errors, fmt.Sprintf("unable to retrieve issue: issue %q is not awaiting page review", k))
		}

		issues = append(issues, i)
	}

	if len(errors) != 0 {
		return nil, fmt.Errorf(strings.Join(errors, "; "))
	}

	return issues, nil
}

func removeIssue(i *models.Issue) error {
	var u = models.SystemUser
	var comment = "Manual issue in page-review removed by administrator"
	var err = i.PrepForRemoval(u.ID, comment)
	if err != nil {
		return fmt.Errorf("unable to prepare issue %d for removal: %w", i.ID, err)
	}

	err = jobs.QueueRemoveErroredIssue(i, conf.ErroredIssuesPath)
	if err != nil {
		return fmt.Errorf("unable to queue errored issue %d for removal: %w", i.ID, err)
	}

	return nil
}
