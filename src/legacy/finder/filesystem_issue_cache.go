package main

import (
	"fileutil"
	"log"
	"path/filepath"
	"time"
)

// cacheAllFilesystemIssues calls all the individual cache functions for the
// myriad of ways we store issue information in the various locations
func cacheAllFilesystemIssues() {
	var err error

	err = cacheSFTPIssues()
	if err != nil {
		log.Fatalf("Error trying to cache SFTPed issues: %s", err)
	}
}

func cacheSFTPIssues() error {
	// First find all titles
	var titlePaths, err = fileutil.FindDirectories(Conf.MasterPDFUploadPath)
	if err != nil {
		return err
	}

	// Find all issues next
	for _, titlePath := range titlePaths {
		// Make sure we have a legitimate title
		var titleName = filepath.Base(titlePath)
		var title = sftpTitlesByName[titleName]
		if title == nil {
			log.Printf("WARNING: Invalid title detected: %s", titleName)
			continue
		}

		// In SFTP-land, issues are ALWAYS subdirectories in the format of
		// YYYY-MM-DD, or else we consider them errors
		var issuePaths, err = fileutil.FindDirectories(titlePath)
		if err != nil {
			return err
		}

		for _, issuePath := range issuePaths {
			var base = filepath.Base(issuePath)
			var dt, err = time.Parse("2006-01-02", base)
			if err != nil {
				continue
			}
			var issue = title.AppendIssue(dt, 1)
			cacheIssue(issue, issuePath)
		}
	}

	return nil
}
