package findhandler

import (
	"fmt"
	"issuefinder"
	"schema"
)

// Issue wraps schema.Issue to provide presentation-specific information needed
// for the issue finder tool
type Issue struct {
	*schema.Issue
	Namespace issuefinder.Namespace
}

// DateEdition returns the issue date and edition in a user-friendly way
func (i *Issue) DateEdition() string {
	return fmt.Sprintf("%s, ed. %d", i.Date.Format("2006-01-02"), i.Edition)
}

// WorkflowStep returns a string telling people a human-friendly explanation of
// where this issue is in the workflow
func (i *Issue) WorkflowStep() string {
	switch i.Namespace {
	case issuefinder.Website:
		return "Production website"
	case issuefinder.SFTPUpload:
		return "SFTP upload folder"
	case issuefinder.AwaitingPageReview:
		return "Awaiting page review"
	case issuefinder.AwaitingMetadataReview:
		return "Awaiting metadata review"
	case issuefinder.PDFsAwaitingDerivatives:
		return "PDF awaiting JP2 / XML derivative generation"
	case issuefinder.ScansAwaitingDerivatives:
		return "TIFF/PDF pair awaiting JP2 / XML derivative generation"
	case issuefinder.ReadyForBatching:
		return "Waiting for next batch generation"
	case issuefinder.BatchedOnDisk:
		return "Batched, but not ingested"
	case issuefinder.MasterBackup:
		return "Master PDF backup"
	case issuefinder.PageBackup:
		return "Batch backup"
	default:
		return "Unknown"
	}
}
