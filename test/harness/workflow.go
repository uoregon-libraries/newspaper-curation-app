package main

import (
	"errors"
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// curateCmd stands in for a curator: it enters dummy metadata for every issue
// awaiting metadata entry and queues the issue for review
type curateCmd struct{}

// Execute implements flags.Commander
func (c *curateCmd) Execute([]string) error {
	return processIssues(schema.WSReadyForMetadataEntry, curate)
}

// reviewCmd stands in for a reviewer: it approves every issue awaiting
// metadata review and queues the jobs that finalize it (generating METS XML),
// after which the issue is ready for batching
type reviewCmd struct{}

// Execute implements flags.Commander
func (c *reviewCmd) Execute([]string) error {
	return processIssues(schema.WSAwaitingMetadataReview, review)
}

// processIssues runs fn, as the sysop user, on every issue in the given
// workflow step
func processIssues(ws schema.WorkflowStep, fn func(*models.User, *models.Issue) error) error {
	var err = connectDB()
	if err != nil {
		return err
	}

	// We get all issues, not just those in ws, so the debug output can say why
	// an issue was skipped
	var all []*models.Issue
	all, err = models.Issues().Fetch()
	if err != nil {
		return fmt.Errorf("finding issues: %w", err)
	}
	var issues []*models.Issue
	for _, i := range all {
		if i.WorkflowStep == ws {
			issues = append(issues, i)
			logger.Debugf("Queueing issue %s", i.Key())
		} else {
			logger.Debugf("Skipping issue %s: workflow step is %q (we want %q)", i.Key(), i.WorkflowStep, ws)
		}
	}
	if len(issues) == 0 {
		logger.Infof("No issues were found in %q", ws)
		return nil
	}

	var u = models.FindActiveUserWithLogin("sysop")
	if u == nil {
		return errors.New(`no active "sysop" user (run the seed command to create one)`)
	}

	for _, i := range issues {
		logger.Infof("Processing %s", i.Key())
		err = i.Claim(u.ID)
		if err == nil {
			err = fn(u, i)
		}
		if err != nil {
			return fmt.Errorf("issue %s: %w", i.Key(), err)
		}
	}
	return nil
}

func curate(u *models.User, i *models.Issue) error {
	i.Issue = "0"
	i.Volume = "0"
	i.EditionLabel = ""
	i.DateAsLabeled = i.Date
	i.Edition = 1
	i.PageLabels = nil

	var pages, err = findFiles(i.Location, ".pdf")
	if err != nil {
		return err
	}
	for range pages {
		i.PageLabels = append(i.PageLabels, "0")
	}

	err = jobs.QueueIssueForMetadataReview(i, u)
	if err != nil {
		return fmt.Errorf("queueing for review: %w", err)
	}
	return nil
}

func review(u *models.User, i *models.Issue) error {
	var err = i.ApproveMetadata(u.ID)
	if err != nil {
		return fmt.Errorf("approving metadata: %w", err)
	}
	err = jobs.QueueFinalizeIssue(i)
	if err != nil {
		return fmt.Errorf("queueing finalization: %w", err)
	}
	return nil
}
