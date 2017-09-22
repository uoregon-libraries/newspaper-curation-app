package main

import (
	"config"
	"db"
	"fileutil"
	"jobs"
	"logger"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var pdfFilenameRegex = regexp.MustCompile(`(?i:^[0-9]{4}.pdf)`)

func scanPageReviewIssues(c *config.Config) {
	logger.Debug("Looking for page-review issues ready to queue for derivative processing")
	var list, err = db.FindIssuesInPageReview()
	if err != nil {
		logger.Error("Unable to query issues in page review: %s", err)
		return
	}

	for _, dbIssue := range list {
		if pagesReady(dbIssue.Location) {
			queueIssueForDerivatives(dbIssue)
		}
	}
}

// pagesReady returns true if all pdf files are in the format of 0000.pdf,
// nothing has been touched in the past hour, and no files exist that aren't
// either PDFs or hidden files.
func pagesReady(location string) bool {
	var infos, err = fileutil.ReaddirSorted(location)
	if err != nil {
		logger.Error("Unable to scan %q for renamed PDFs: %s", location, err)
		return false
	}

	for _, info := range infos {
		var fName = info.Name()

		// Ignore hidden files
		if filepath.Base(fName)[0] == '.' {
			logger.Debug("Ignoring hidden file %q", fName)
			continue
		}

		// Failure to match regex isn't worth logging; it just means the page
		// reviewers may not be done renaming
		if !pdfFilenameRegex.MatchString(fName) {
			logger.Debug("Not processing %q (%q doesn't match rename regex)", location, fName)
			return false
		}

		// If any PDF was touched less than an hour ago, we don't consider it safe
		// to process yet
		if time.Since(info.ModTime()) < time.Hour {
			logger.Debug("Not processing %q (%q was touched too recently)", location, fName)
			return false
		}
	}

	return true
}

// queueIssueForDerivatives first renames the directory so no more
// modifications are likely to take place, then moves the PDFs (and only the
// PDFs) to the workflow directory for derivative processing
func queueIssueForDerivatives(dbIssue *db.Issue) {
	var oldDir = dbIssue.Location
	var newDir = filepath.Join(filepath.Dir(oldDir), ".notouchie-"+filepath.Base(oldDir))
	logger.Info("Renaming %q to %q to prepare for derivative processing", oldDir, newDir)
	var err = os.Rename(oldDir, newDir)
	if err != nil {
		logger.Error("Unable to rename %q for derivative processing: %s", oldDir, err)
		return
	}
	dbIssue.Location = newDir
	dbIssue.AwaitingPageReview = false
	err = dbIssue.Save()
	if err != nil {
		logger.Critical("Unable to update db Issue (location and awaiting page review status): %s", err)
		return
	}

	// Queue up move to workflow dir
	jobs.QueueMoveIssueForDerivatives(dbIssue, newDir)
}
