package main

import (
	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
	"github.com/uoregon-libraries/newspaper-curation-app/src/uploads"
)

func scanScannerIssues(c *config.Config) {
	logger.Infof("scanner-scanner: checking for in-house digitizations ready to move into the workflow")
	var scanner = issuewatcher.NewScanner(c)

	logger.Debugf("scanner-scanner: reading issues - this may take a few minutes")
	var err = scanner.Scan()
	if err != nil {
		logger.Criticalf("scanner-scanner: unable to read issues: %s", err)
		return
	}
	logger.Debugf("scanner-scanner: done reading issues")

	for _, issue := range scanner.Finder.Issues {
		if issue.WorkflowStep != schema.WSScan {
			continue
		}

		var i = uploads.New(issue, scanner, c)
		i.ValidateAll()
		if i.Errors.Major().Len() != 0 {
			var errs []string
			for _, err := range i.Errors.Major().All() {
				errs = append(errs, err.Message())
			}
			logger.Debugf("scanner-scanner: skipping issue %q: %#v", i.Key(), errs)
			continue
		}

		logger.Infof("scanner-scanner: moving issue %q into NCA", i.Key())
		var err = i.Queue()
		if err != nil {
			logger.Warnf("scanner-scanner: skipping %q: %s", i.Key(), err)
		}
	}
}
