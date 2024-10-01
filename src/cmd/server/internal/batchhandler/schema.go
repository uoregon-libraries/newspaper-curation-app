package batchhandler

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// Batch wraps models.Batch to decorate for template use
type Batch struct {
	*models.Batch
	FlaggedIssues   []*models.FlaggedIssue
	UnflaggedIssues []*models.Issue
	Issues          []*models.Issue
	Actions         []*models.Action
	PageCount       int
	cv              *CanValidation
}

func wrapBatch(batch *models.Batch, currentUser *models.User) (*Batch, error) {
	var err error
	var b = &Batch{Batch: batch}
	b.cv = Can(currentUser, b)
	b.Issues, err = b.Batch.Issues()
	if err != nil {
		return nil, fmt.Errorf("fetching batch %d (%q) issues: %w", b.Batch.ID, b.Batch.Name, err)
	}

	b.FlaggedIssues, err = b.Batch.FlaggedIssues()
	if err != nil {
		return nil, fmt.Errorf("fetching batch %d (%q) flagged issues: %w", b.Batch.ID, b.Batch.Name, err)
	}

	var isFlagged = make(map[string]bool)
	for _, i := range b.FlaggedIssues {
		isFlagged[i.Issue.Key()] = true
	}

	// Compute page count as well as creating the unflagged issues list
	for _, i := range b.Issues {
		b.PageCount += i.PageCount
		if !isFlagged[i.Key()] {
			b.UnflaggedIssues = append(b.UnflaggedIssues, i)
		}
	}

	b.Actions, err = b.Batch.Actions()
	if err != nil {
		return nil, fmt.Errorf("fetching batch %d (%q) actions: %w", b.Batch.ID, b.Batch.Name, err)
	}

	return b, nil
}

func wrapBatches(list []*models.Batch, currentUser *models.User) ([]*Batch, error) {
	var err error
	var batches = make([]*Batch, len(list))
	for i, b := range list {
		batches[i], err = wrapBatch(b, currentUser)
		if err != nil {
			return nil, fmt.Errorf("wrapping batches: %w", err)
		}
	}

	return batches, nil
}

// ReadyForQC is true if the batch is awaiting a quality control check
func (b *Batch) ReadyForQC() bool {
	return b.Status == models.BatchStatusQCReady
}

// ReadyForArchive is true if the batch has gone live and has been moved to the
// archival destination (dark-archive xfer location for UO, but could be the
// final archive location for others)
func (b *Batch) ReadyForArchive() bool {
	// This status isn't set until *after* the batch is moved out of NCA, so we
	// can rely on the status alone here
	return b.Status == models.BatchStatusLive
}

// ReadyForFlaggingIssues is true if the batch is ready for a batch reviewer to
// flag which issues need to be removed
func (b *Batch) ReadyForFlaggingIssues() bool {
	return b.Status == models.BatchStatusQCFlagIssues
}

// Can returns our CanValidation data for the currently logged in user and this
// batch so we aren't asking for globals in the HTML template just to check
// permissions
func (b *Batch) Can() *CanValidation {
	return b.cv
}
