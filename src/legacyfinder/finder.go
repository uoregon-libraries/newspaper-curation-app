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

	err = f.findStandardIssues(realFinder)
	if err != nil {
		return nil, fmt.Errorf("unable to cache standard filesystem issues: %s", err)
	}

	err = realFinder.FindDiskBatches(f.config.BatchOutputPath)
	if err != nil {
		return nil, fmt.Errorf("unable to cache batches: %s", err)
	}
	return realFinder, nil
}

// findStandardIssues deals with all the various locations for issues which
// are not in a batch directory structure.  This doesn't mean they haven't been
// batched, just that the directory uses the somewhat consistent pdf-to-chronam
// structure `topdir/sftpnameOrLCCN/yyyy-mm-dd/`
func (f *Finder) findStandardIssues(realFinder *issuefinder.Finder) error {
	var locs = []string{
		f.config.MasterPDFBackupPath,
		f.config.PDFPageReviewPath,
		f.config.PDFPagesAwaitingMetadataReview,
		f.config.PDFIssuesAwaitingDerivatives,
		f.config.ScansAwaitingDerivatives,
		f.config.PDFPageBackupPath,
		f.config.PDFPageSourcePath,
	}

	var namespaces = map[string]issuefinder.Namespace{
		f.config.MasterPDFBackupPath:            issuefinder.MasterBackup,
		f.config.PDFPageReviewPath:              issuefinder.AwaitingPageReview,
		f.config.PDFPagesAwaitingMetadataReview: issuefinder.AwaitingMetadataReview,
		f.config.PDFIssuesAwaitingDerivatives:   issuefinder.PDFsAwaitingDerivatives,
		f.config.ScansAwaitingDerivatives:       issuefinder.ScansAwaitingDerivatives,
		f.config.PDFPageBackupPath:              issuefinder.PageBackup,
		f.config.PDFPageSourcePath:              issuefinder.ReadyForBatching,
	}

	for _, loc := range locs {
		var err = realFinder.FindStandardIssues(namespaces[loc], loc)
		if err != nil {
			return err
		}
	}

	return nil
}
