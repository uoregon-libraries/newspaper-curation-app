package batchhandler

import "github.com/uoregon-libraries/newspaper-curation-app/src/models"

// Batch wraps models.Batch to decorate for template use
type Batch struct {
	*models.Batch
}

func wrapBatch(b *models.Batch) *Batch {
	return &Batch{b}
}

func wrapBatches(list []*models.Batch) []*Batch {
	var batches = make([]*Batch, len(list))
	for i, b := range list {
		batches[i] = wrapBatch(b)
	}

	return batches
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
