package models

import (
	"fmt"
	"hash/crc32"
	"time"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
)

// These are all possible batch status values
const (
	BatchStatusDeleted   = "deleted"    // Batch wasn't fixable and had to be removed
	BatchStatusPending   = "pending"    // Not yet built or in the process of being built
	BatchStatusQCReady   = "qc_ready"   // Ready for ingest onto staging
	BatchStatusOnStaging = "on_staging" // On the staging server awaiting QC
	BatchStatusFailedQC  = "failed_qc"  // On staging, but QC failed it; it needs to be pulled and fixed
	BatchStatusPassedQC  = "passed_qc"  // On staging, passed QC; it needs to be pulled from staging and pushed live
	BatchStatusLive      = "live"       // Batch has gone live; batch and its issues need to be archived
	BatchStatusLiveDone  = "live_done"  // Batch has gone live; batch and its issues have been archived and are no longer on the filesystem
)

// Batch contains metadata for generating a batch XML.  Issues can be
// associated with a single batch, and a batch will typically have many issues
// assigned to it.
type Batch struct {
	ID          int `sql:",primary"`
	MARCOrgCode string
	Name        string
	CreatedAt   time.Time
	ArchivedAt  time.Time
	Status      string
	Location    string

	issues []*Issue
}

// FindBatch looks for a batch by its id
func FindBatch(id int) (*Batch, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var b = &Batch{}
	var ok = op.Select("batches", b).Where("id = ?", id).First(b)
	if !ok {
		return nil, op.Err()
	}
	return b, op.Err()
}

// InProcessBatches returns the full list of in-process batches (not live, not pending)
func InProcessBatches() ([]*Batch, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug

	var list []*Batch
	op.Select("batches", &Batch{}).Where(
		"status IN (?, ?, ?, ?)",
		BatchStatusQCReady, BatchStatusOnStaging, BatchStatusFailedQC, BatchStatusPassedQC,
	).AllObjects(&list)

	return list, op.Err()
}

// FindLiveArchivedBatches returns all batches that are still live, but have an
// archived_at value
func FindLiveArchivedBatches() ([]*Batch, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug

	var list []*Batch
	op.Select("batches", &Batch{}).
		Where("status = ? AND archived_at > ?", BatchStatusLive, time.Time{}).
		AllObjects(&list)

	return list, op.Err()
}

// CreateBatch creates a batch in the database, using its ID combined with the
// hash of the site's web root string to generate a unique batch name, and
// associating the given list of issues.  This is inefficient, but it gets the
// job done.
//
// Background: the batch name is deterministic because we need to be sure we
// don't reuse the various components (e.g., "Jade", "Pine", "Maple", etc.) too
// frequently.  But if NCA is used for two different sites, batch names really
// shouldn't be exactly the same.  Adding the CRC32 of the webroot string
// ensures that we stick with a sequence, keeping collisions unlikely, but a
// different site would have a totally different sequence.
func CreateBatch(webroot, moc string, issues []*Issue) (*Batch, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	var b = &Batch{MARCOrgCode: moc, CreatedAt: time.Now(), issues: issues, Status: BatchStatusPending}
	var err = b.SaveOp(op)
	if err != nil {
		return nil, err
	}

	for _, i := range issues {
		i.BatchID = b.ID
		i.SaveOp(op)
	}

	var chksum = crc32.ChecksumIEEE([]byte(webroot))
	b.Name = RandomBatchName(uint32(b.ID) + chksum)
	err = b.SaveOp(op)
	return b, err
}

// Issues pulls all issues from the database which have this batch's ID
func (b *Batch) Issues() ([]*Issue, error) {
	if len(b.issues) > 0 {
		return b.issues, nil
	}

	if b.ID == 0 {
		return b.issues, nil
	}

	var issues, err = FindIssuesByBatchID(b.ID)
	b.issues = issues
	return b.issues, err
}

// FullName returns the name of a batch as it is needed for chronam / ONI.
//
// Note that currently we assume all generated batches will be _ver01, because
// we would usually generate a completely new batch if one were in such a state
// as to need to be pulled from production.
func (b *Batch) FullName() string {
	return fmt.Sprintf("batch_%s_%s%s_ver01", b.MARCOrgCode, b.CreatedAt.Format("20060102"), b.Name)
}

// AwardYear uses the batch creation date to produce the "award year" - this is
// the most similar value we can produce
func (b *Batch) AwardYear() int {
	return b.CreatedAt.Year()
}

// Save creates or updates the Batch in the batches table
func (b *Batch) Save() error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	return b.SaveOp(op)
}

// SaveOp saves the batch to the batches table with a custom operation for
// easier transactions
func (b *Batch) SaveOp(op *magicsql.Operation) error {
	op.Save("batches", b)
	return op.Err()
}

// Delete removes all issues from this batch and sets its status to "deleted".
// Caller must clean up the filesystem.
func (b *Batch) Delete() error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	b.Status = BatchStatusDeleted
	b.Location = ""
	var err = b.SaveOp(op)
	if err != nil {
		return err
	}

	var issues []*Issue
	issues, err = b.Issues()
	if err != nil {
		return err
	}

	for _, i := range issues {
		i.BatchID = 0
		i.SaveOp(op)
	}
	return op.Err()
}

// Close finalizes a batch that's live and archived by setting its status to
// BatchStatusLiveDone.  This has some of our "safety first" business logic you
// don't get if you close the batch manually, e.g., it must be in the "live"
// status and it must have been archived at least four weeks ago.
func (b *Batch) Close() error {
	if b.Status != BatchStatusLive {
		return fmt.Errorf("cannot close batch unless its status is live")
	}
	var fourWeeksAgo = time.Now().Add(-time.Hour * 24 * 7 * 4)
	if !b.ArchivedAt.Before(fourWeeksAgo) {
		return fmt.Errorf("cannot close live batches archived fewer than four weeks ago")
	}

	b.Status = BatchStatusLiveDone
	return b.Save()
}
