package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

var pdfFilenameRegex = regexp.MustCompile(`(?i:^[0-9]+.pdf)`)
var notRenamedRegex = regexp.MustCompile(`(?i:^seq-[0-9]{4}.pdf)`)

// pageReviewIssueReady tells us if issues being manually modified are ready to
// be moved on in the workflow.  Pages are considered ready if all of the
// following are true:
//
// * The path hasn't been modified less than minAge ago
// * No files exist that don't match the pdf regex
//   * Dotfiles are the exception, as we simply ignore these since Bridge and Macs drop these everywhere
// * Nothing is in the path that was modified less than minAge ago
func pageReviewIssueReady(path string, minAge time.Duration) bool {
	// Validate that the directory itself can be statted and hasn't been touched
	// in at least an hour
	var info, err = os.Stat(path)
	if err != nil {
		logger.Errorf("Unable to read %q: %s", path, err)
		return false
	}
	if time.Since(info.ModTime()) < minAge {
		return false
	}

	// Gather info on all items in the issue path
	var infos []os.FileInfo
	infos, err = ioutil.ReadDir(path)
	if err != nil {
		logger.Errorf("Unable to scan %q for renamed PDFs: %s", path, err)
		return false
	}

	for _, info := range infos {
		var fName = info.Name()

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

		// If any file was touched less than an hour ago, we don't consider it safe
		// to process yet
		if time.Since(info.ModTime()) < minAge {
			return false
		}
	}

	return true
}
