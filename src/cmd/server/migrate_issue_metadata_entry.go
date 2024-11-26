package main

import (
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

func migrateIssuesMissingMetadataEntry() {
	var issues, err = models.FindIssuesLackingMetadataEntryDate()
	if err != nil {
		logger.Fatalf("Unable to search for legacy issues to migrate: %s", err)
	}

	if len(issues) == 0 {
		return
	}

	logger.Infof("Converting legacy issues lacking metadata entry date... this may take several minutes.")
	for _, issue := range issues {
		// To find out when metadata was entered (if it was in fact entered), we have
		// to reverse the actions list to find the most recent metadata entry time
		migrateIssueMetadataEnteredAt(issue)
	}
	logger.Infof("Legacy issues lacking metadata entry date have been migrated.")
}

func migrateIssueMetadataEnteredAt(issue *models.Issue) {
	var actions = issue.AllWorkflowActions()
	for x := len(actions) - 1; x >= 0; x-- {
		if actions[x].Type() == models.ActionTypeMetadataEntry {
			issue.MetadataEnteredAt = actions[x].CreatedAt
			var err = issue.SaveWithoutAction()
			if err != nil {
				logger.Fatalf("Unable to migrate legacy issues: %s", err)
			}
			return
		}
	}
	issue.MetadataEnteredAt = time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	var err = issue.SaveWithoutAction()
	if err != nil {
		logger.Fatalf("Unable to migrate legacy issues: %s", err)
	}
}
