package main

import (
	"fmt"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/uploads"
)

// loadIssuesCmd replaces NCA's workflow tree with the test issues (see
// copySourcesCmd), backdates them so NCA won't consider them new, and queues
// the born-digital ones for page review, the way a curator would. Scanned
// issues don't need queueing: NCA picks them up on its own, since in-house
// scans are assumed to be built to spec.
//
// NCA's job runner mustn't be running: it would start on issues while
// they're still being copied.
type loadIssuesCmd struct {
	copySourcesCmd
}

// Execute implements flags.Commander
func (c *loadIssuesCmd) Execute(args []string) error {
	var err = c.copySourcesCmd.Execute(args)
	if err == nil {
		// A month is old enough that no setting considers the issues new
		var older = &makeOlderCmd{Days: 30}
		err = older.Execute(nil)
	}
	if err == nil {
		err = queueBornDigital()
	}
	return err
}

// queueBornDigital queues every valid born-digital upload for page review,
// the same way bulk-issue-queue does. Invalid issues are skipped with a
// warning, since the test data deliberately has some.
func queueBornDigital() error {
	var err = connectDB()
	if err != nil {
		return err
	}

	var uploaded = issuewatcher.NewScanner(conf).DisableDB().DisableWeb().DisableScannedUpload()
	err = uploaded.Scan()
	if err != nil {
		return fmt.Errorf("scanning born-digital uploads: %w", err)
	}

	// Validation needs everything NCA knows about, to catch duplicates
	logger.Infof("Reading issue data: this takes a long time if the issue cache hasn't been built")
	var everything = issuewatcher.NewScanner(conf)
	err = everything.Scan()
	if err != nil {
		return fmt.Errorf("scanning issues: %w", err)
	}

	var queued int
	for _, issue := range uploaded.Finder.Issues {
		var i = uploads.New(issue, everything, conf)
		i.ValidateAll()
		if i.Errors.Len() != 0 {
			var messages []string
			for _, e := range i.Errors.All() {
				messages = append(messages, e.Message())
			}
			logger.Warnf("Skipping %q: %s", i.Key(), strings.Join(messages, "; "))
			continue
		}

		err = i.Queue()
		if err != nil {
			logger.Warnf("Skipping %q: %s", i.Key(), err)
			continue
		}
		queued++
	}

	logger.Infof("Queued %d of %d born-digital issues", queued, len(uploaded.Finder.Issues))
	return nil
}
