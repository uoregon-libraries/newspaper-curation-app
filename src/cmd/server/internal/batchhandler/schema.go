package batchhandler

import (
	"fmt"
	"path/filepath"

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
}

func wrapBatch(batch *models.Batch) (*Batch, error) {
	var err error
	var b = &Batch{Batch: batch}
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

func wrapBatches(list []*models.Batch) ([]*Batch, error) {
	var err error
	var batches = make([]*Batch, len(list))
	for i, b := range list {
		batches[i], err = wrapBatch(b)
		if err != nil {
			return nil, fmt.Errorf("wrapping batches: %w", err)
		}
	}

	return batches, nil
}

// Unavailable returns true if the batch status indicates it's not currently
// able to be acted upon by users (doesn't need action)
func (b *Batch) Unavailable() bool {
	return !b.StatusMeta.NeedsAction
}

// ReadyForStaging is true if the batch is ready to be loaded onto staging
func (b *Batch) ReadyForStaging() bool {
	return b.Status == models.BatchStatusStagingReady
}

// ReadyForProduction is true if the batch has passed QC
func (b *Batch) ReadyForProduction() bool {
	return b.Status == models.BatchStatusPassedQC
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

// LiveLocation returns the location the batch is stored on production, which
// is necessary for instructions that involve loading a batch somewhere
func (b *Batch) LiveLocation() string {
	return filepath.Join(conf.BatchProductionPath, b.FullName())
}
