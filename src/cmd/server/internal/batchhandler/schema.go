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

// ReadyForQC is true if the batch is awaiting a quality control check
func (b *Batch) ReadyForQC() bool {
	return b.Status == models.BatchStatusQCReady
}
