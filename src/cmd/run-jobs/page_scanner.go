package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

func scanPageReviewIssues(c *config.Config) {
	var list, err = models.FindIssuesInPageReview()
	if err != nil {
		logger.Errorf("Unable to query issues in page review: %s", err)
		return
	}

	for _, dbIssue := range list {
		if pageReviewIssueReady(dbIssue.Location, time.Hour) {
			queueIssueForDerivatives(dbIssue, c.WorkflowPath)
		}
	}
}

// queueIssueForDerivatives first renames the directory so no more
// modifications are likely to take place, then queues the directory for being
// moved to the workflow space
func queueIssueForDerivatives(dbIssue *models.Issue, workflowPath string) {
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
	jobs.QueueMoveIssueForDerivatives(dbIssue, workflowPath)
}
