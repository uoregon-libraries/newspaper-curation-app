// Package legacyfinder is our legacy issue finder.  It acts as a specialized
// layer on top of the generic issuefinder to include behaviors necessary for
// finding issues from all known locations by reading our settings file and
// running the appropriate searches.
//
// This is considered legacy despite being new code because long-term we want
// the vast majority of issue workflow to be data-driven, not
// filesystem-path-driven.
package legacyfinder

import (
	"config"
	"fmt"
	"issuefinder"
)

// Finder wraps issuefinder.Finder with some legacy rules and data as well as a
// global lock so we can refresh the cache with thread-safety
type Finder struct {
	finder  *issuefinder.Finder
	config  *config.Config
	webroot string
	tempdir string
}

// FindIssues calls all the individual find* functions for the myriad of ways
// we store issue information in the various locations, returning the
// issuefinder.Finder with all this data.  Since this operates independently,
// creating and returning a new issuefinder, it's threadsafe and can be called
// with or without the issue watcher loop.
func (f *Finder) FindIssues() (*issuefinder.Finder, error) {
	var realFinder = issuefinder.New()
	var err error

	err = realFinder.FindWebBatches(f.webroot, f.tempdir)
	if err != nil {
		return nil, fmt.Errorf("unable to cache web batches: %s", err)
	}

	err = realFinder.FindSFTPIssues(f.config.MasterPDFUploadPath)
	if err != nil {
		return nil, fmt.Errorf("unable to cache sftp issues: %s", err)
	}
	return realFinder, nil
}
