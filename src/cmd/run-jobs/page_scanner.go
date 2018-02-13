package main

import (
	"config"
	"db"
	"jobs"
	"os"
	"path/filepath"
	"regexp"
	"schema"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
)

var pdfFilenameRegex = regexp.MustCompile(`(?i:^[0-9]{4}.pdf)`)

func scanPageReviewIssues(c *config.Config) {
	var list, err = db.FindIssuesInPageReview()
	if err != nil {
		logger.Errorf("Unable to query issues in page review: %s", err)
		return
	}

	for _, dbIssue := range list {
		if issuePagesReady(dbIssue.Location, time.Hour, pdfFilenameRegex) {
			queueIssueForDerivatives(dbIssue)
		}
	}
}

// queueIssueForDerivatives first renames the directory so no more
// modifications are likely to take place, then queues the directory for being
// moved to the workflow space
func queueIssueForDerivatives(dbIssue *db.Issue) {
	var oldDir = dbIssue.Location
	var newDir = filepath.Join(filepath.Dir(oldDir), ".notouchie-"+filepath.Base(oldDir))
	logger.Infof("Renaming %q to %q to prepare for derivative processing", oldDir, newDir)
	var err = os.Rename(oldDir, newDir)
	if err != nil {
		logger.Errorf("Unable to rename %q for derivative processing: %s", oldDir, err)
		return
	}
	dbIssue.Location = newDir
	dbIssue.WorkflowStep = schema.WSAwaitingProcessing
	err = dbIssue.Save()
	if err != nil {
		logger.Criticalf("Unable to update db Issue (location and awaiting page review status): %s", err)
		return
	}

	// Queue up move to workflow dir
	jobs.QueueMoveIssueForDerivatives(dbIssue, newDir)
}
