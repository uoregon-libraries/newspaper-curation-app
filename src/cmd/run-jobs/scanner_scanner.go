package main

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
	"github.com/uoregon-libraries/newspaper-curation-app/src/uploads"
)

func scanScannerIssues(c *config.Config) {
	logger.Infof("scanner-scanner: checking for in-house digitizations ready to move into the workflow")
	var scanner = issuewatcher.NewScanner(c)
	var err = scanner.Scan()
	if err != nil {
		logger.Criticalf("scanner-scanner: unable to read issues: %s", err)
		return
	}

	for _, issue := range scanner.Finder.Issues {
		if issue.WorkflowStep != schema.WSScan {
			continue
		}

		var i = uploads.New(issue, scanner, c)
		i.ValidateAll()
		if len(i.Errors) != 0 {
			continue
		}

		logger.Infof("scanner-scanner: moving issue %q into NCA", i.Key())
		var err = i.Queue()
		if err != nil {
			logger.Warnf("scanner-scanner: skipping %q: %s", i.Key(), err)
		}
	}
}
