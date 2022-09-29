package issuewatcher

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuefinder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// Scanner sets up all the necessary data to run issuefinders across all our
// standard locations.  By default, a Scan() call won't do anything - one or
// more of the EnableXXX methods must first be called to set up paths.
type Scanner struct {
	Finder              *issuefinder.Finder
	Webroot             string
	Tempdir             string
	ScanUpload          string
	PDFUpload           string
	PDFBatchMARCOrgCode string
	Lookup              *schema.Lookup
	CanonIssues         map[string]*schema.Issue

	skipweb  bool
	skipsftp bool
	skipscan bool
	skipdb   bool
}

// newScanner initializes data not related to the app configuration
func newScanner() *Scanner {
	return &Scanner{
		Finder:      issuefinder.New(),
		CanonIssues: make(map[string]*schema.Issue),
	}
}

// NewScanner sets up the Scanner with no data
func NewScanner(conf *config.Config) *Scanner {
	var s = newScanner()
	s.Webroot = conf.NewsWebroot
	s.Tempdir = conf.IssueCachePath
	s.ScanUpload = conf.ScanUploadPath
	s.PDFUpload = conf.PDFUploadPath
	s.PDFBatchMARCOrgCode = conf.PDFBatchMARCOrgCode

	return s
}

// DisableWeb sets the flag to skip web searches
func (s *Scanner) DisableWeb() *Scanner {
	s.skipweb = true
	return s
}

// DisableSFTPUpload sets the flag to skip sftp upload searches
func (s *Scanner) DisableSFTPUpload() *Scanner {
	s.skipsftp = true
	return s
}

// DisableScannedUpload sets the flag to skip scanned upload searches
func (s *Scanner) DisableScannedUpload() *Scanner {
	s.skipscan = true
	return s
}

// DisableDB sets the flag to skip database searches
func (s *Scanner) DisableDB() *Scanner {
	s.skipdb = true
	return s
}

// Duplicate creates a new Scanner with the same configuration as this one, but
// with no data
func (s *Scanner) Duplicate() *Scanner {
	var s2 = newScanner()
	s2.Webroot = s.Webroot
	s2.Tempdir = s.Tempdir
	s2.ScanUpload = s.ScanUpload
	s2.PDFUpload = s.PDFUpload
	s2.skipweb = s.skipweb
	s2.skipsftp = s.skipsftp
	s2.skipscan = s.skipscan
	s2.skipdb = s.skipdb

	return s2
}

// LookupIssues returns a list of schema Issues for the give search key
func (s *Scanner) LookupIssues(key *schema.Key) []*schema.Issue {
	return s.Lookup.Issues(key)
}

// Scan calls all the individual find* functions for the myriad of ways we
// store issue information in the various locations (dependent on what's been
// enabled).  The Scanner's issuefinder is replaced only after successful
// searching to ensure minimal disruption, especially in the event of an error.
func (s *Scanner) Scan() error {
	var f = issuefinder.New()
	var err error

	if !s.skipweb {
		_, err = f.FindWebBatches(s.Webroot, s.Tempdir)
		if err != nil {
			return fmt.Errorf("unable to cache web batches: %w", err)
		}
	}

	if !s.skipdb {
		_, err = f.FindInProcessIssues()
		if err != nil {
			return fmt.Errorf("unable to cache in-process issues: %w", err)
		}
	}

	if !s.skipsftp {
		_, err = f.FindSFTPIssues(s.PDFUpload, s.PDFBatchMARCOrgCode)
		if err != nil {
			return fmt.Errorf("unable to cache sftp issues: %w", err)
		}
	}

	if !s.skipscan {
		_, err = f.FindScannedIssues(s.ScanUpload)
		if err != nil {
			return fmt.Errorf("unable to cache scanned issues: %w", err)
		}
	}

	// Create a new lookup using the new finder's data
	s.Lookup = schema.NewLookup()
	s.Lookup.Populate(f.Issues)

	// If this is an "everything" scanner, we need to check all issues for dupes
	if !s.skipweb && !s.skipdb && !s.skipsftp && !s.skipscan {
		for _, i := range f.Issues {
			i.CheckDupes(s.Lookup)
		}
	}

	// Swap out the finder
	s.Finder = f

	return nil
}
