package main

import (
	"fileutil"
	"log"
	"path/filepath"
	"time"
)

// issueMap links a textual issue key to the Issue object
type issueMap map[string][]*Issue

// issueLocMap links a textual issue key to all known issue locations
type issueLocMap map[string][]string

// titleLookup lets us find titles by LCCN
var titleLookup = make(map[string]*Title)

// issueLookup lets us find issues by key
var issueLookup = make(issueMap)

// issueLocLookup lets us find an issue's raw location
var issueLocLookup = make(issueLocMap)

// cacheIssue shortcuts the process of getting an issue's key and storing its
// filesystem path in issueLocLookup and the object in issueLookup
func cacheIssue(i *Issue, location string) {
	var k = i.Key()
	var iList = issueLookup[k]
	iList = append(iList, i)
	issueLookup[k] = iList

	var ipList = issueLocLookup[k]
	ipList = append(ipList, location)
	issueLocLookup[k] = ipList
}

// cacheAllIssues calls all the individual cache functions for the myriad of
// ways we store issue information in the various locations
func cacheAllIssues() {
	var err error

	err = cacheSFTPIssues()
	if err != nil {
		log.Fatalf("Error trying to cache SFTPed issues: %s", err)
	}

	// TODO: Still need to handle (or decide not to bother handling):
	// - config.PDFIssuesAwaitingDerivatives
	// - config.PDFPageReviewPath
	// - config.PDFPagesAwaitingMetadataReview
	// - config.PDFPageSourcePath
	// - config.BatchOutputPath
	// - config.ScansAwaitingDerivatives
	// - config.MasterPDFBackupPath
	// - config.PDFPageBackupPath
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
