package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/lastmod"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

var pdfFilenameRegex = regexp.MustCompile(`(?i:^[0-9]+.pdf)`)
var notRenamedRegex = regexp.MustCompile(`(?i:^seq-[0-9]{4}.pdf)`)

// pageReviewIssueReady tells us if issues being manually modified are ready to
// be moved on in the workflow.  Pages are considered ready if all of the
// following are true:
//
// * The central last-modified check returns a value that's at least minAge ago
// * No files exist that don't match the pdf regex
//   - Dotfiles are the exception, as we simply ignore these since Bridge and Macs drop these everywhere
func pageReviewIssueReady(path string, minAge time.Duration) bool {
	var t, err = lastmod.Time(path)
	if err != nil {
		logger.Errorf("Unable to check last mod time for %q: %s", path, err)
		return false
	}

	if time.Since(t) < minAge {
		return false
	}

	// Gather info on all items in the issue path
	var entries []fs.DirEntry
	entries, err = os.ReadDir(path)
	if err != nil {
		logger.Errorf("Unable to scan %q for renamed PDFs: %s", path, err)
		return false
	}

	// Check that the filenames are all valid
	for _, entry := range entries {
		var fName = entry.Name()

		// Ignore hidden files
		if filepath.Base(fName)[0] == '.' {
			continue
		}

		// Ignore thumbs.db
		if strings.ToLower(fName) == "thumbs.db" {
			continue
		}

		// If files aren't renamed yet, we don't need to log it, just return false
		if notRenamedRegex.MatchString(fName) {
			return false
		}

		// Files that aren't in the "seq-0001.pdf" pattern *and* not in the proper
		// pattern are considered problems
		if !pdfFilenameRegex.MatchString(fName) {
			logger.Errorf("Not processing %q (%q doesn't match valid file regex)", path, fName)
			return false
		}
	}

	return true
}
