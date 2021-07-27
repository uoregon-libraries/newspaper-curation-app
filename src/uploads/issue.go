package uploads

import (
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// DaysIssueConsideredDangerous is how long we require an issue to be untouched
// prior to anybody queueing it
const DaysIssueConsideredDangerous = 2

// Issue wraps a schema.Issue to add upload- and queue-specific validations and
// behavior
type Issue struct {
	*schema.Issue
	scanner       *issuewatcher.Scanner
	conf          *config.Config
	validatedFast bool // true if we checked the "fast" validations already
	validatedAll  bool // true if we checked all validations already
	Files         []*File
}

// New returns an Issue ready for validation checks and queueing.  It requires
// a Scanner for checking dupes as well as global configuration in order to
// know scanned-issue DPI and born-digital MARC organization code.
func New(si *schema.Issue, s *issuewatcher.Scanner, c *config.Config) *Issue {
	var i2 = &Issue{Issue: si, scanner: s, conf: c, Files: make([]*File, len(si.Files))}
	for i, f := range si.Files {
		i2.Files[i] = &File{f}
	}

	return i2
}

// ValidateFast runs all the upload-queue-specific validations except those
// which are slow (namely the DPI check).  This should be run against every
// issue being considered for queue.
func (i *Issue) ValidateFast() {
	// Only validate if we haven't already done so *or* if we had no prior
	// errors.  Validating twice when we had no errors previously can be useful
	// for getting real-time checks.  This is good!  But validating twice when we
	// already have errors can end up giving us duplicate errors.  This... is not
	// quite so good.
	if i.validatedFast && i.Errors.Major().Len() > 0 {
		return
	}
	i.validatedFast = true

	var hrs = 24 * DaysIssueConsideredDangerous
	if time.Since(i.LastModified()) < time.Hour*time.Duration(hrs) {
		i.ErrTooNew(hrs)
	}
	if i.Title.Errors.Major().Len() > 0 {
		i.ErrBadTitle()
	}
	i.CheckDupes(i.scanner.Lookup)
}

// ValidateAll runs through all upload-queue-specific validations and adds
// errors which are only relevant to these issues.  This validator runs the DPI
// check and is therefore fairly slow for scanned uploads.  It shouldn't be run
// in bulk across a large number of issues.
func (i *Issue) ValidateAll() {
	i.ValidateFast()

	// Only validate if we haven't already done so *or* if we had no prior
	// errors.  Validating twice when we had no errors previously can be useful
	// for getting real-time checks.  This is good!  But validating twice when we
	// already have errors can end up giving us duplicate errors.  This... is not
	// quite so good.
	if i.validatedAll && i.Errors.Major().Len() > 0 {
		return
	}
	i.validatedAll = true

	if i.WorkflowStep == schema.WSScan {
		for _, f := range i.Files {
			f.ValidateDPI(i.conf.ScannedPDFDPI)
		}
	}
}
