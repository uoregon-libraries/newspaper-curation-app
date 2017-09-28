package main

import (
	"fileutil"
	"logger"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

// issuePagesReady tells us if issues being manually modified are ready to be
// moved on in the workflow.  Pages are considered ready if all of the
// following are true:
//
// * The path hasn't been modified less than minAge ago
// * No files exist that don't match one of the validFiles regexes
//   * Dotfiles are the exception, as we simply ignore these since Bridge and Macs drop these everywhere
// * Nothing is in the path that was modified less than minAge ago
func issuePagesReady(path string, minAge time.Duration, validFileRegexes ...*regexp.Regexp) bool {
	// Validate that the directory itself can be statted and hasn't been touched
	// in at least an hour
	var info, err = os.Stat(path)
	if err != nil {
		logger.Error("Unable to read %q: %s", path, err)
		return false
	}
	if time.Since(info.ModTime()) < minAge {
		logger.Debug("Not processing %q (directory was touched too recently)", path)
		return false
	}

	// Gather info on all items in the issue path
	var infos []os.FileInfo
	infos, err = fileutil.ReaddirSorted(path)
	if err != nil {
		logger.Error("Unable to scan %q for renamed PDFs: %s", path, err)
		return false
	}

	for _, info := range infos {
		var fName = info.Name()

		// Ignore hidden files
		if filepath.Base(fName)[0] == '.' {
			logger.Debug("Ignoring hidden file %q", fName)
			continue
		}

		// Failure to match one of the regexes isn't an error; it just means people may not be
		// done working on the issue files
		var matchesOneRegex = false
		for _, validFileRegex := range validFileRegexes {
			if validFileRegex.MatchString(fName) {
				matchesOneRegex = true
				continue
			}
		}
		if !matchesOneRegex {
			logger.Debug("Not processing %q (%q doesn't match valid file regex)", path, fName)
			return false
		}

		// If any file was touched less than an hour ago, we don't consider it safe
		// to process yet
		if time.Since(info.ModTime()) < minAge {
			logger.Debug("Not processing %q (%q was touched too recently)", path, fName)
			return false
		}
	}

	return true
}
