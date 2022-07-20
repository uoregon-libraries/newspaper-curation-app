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
	BatchStatusDeleted      = "deleted"       // Batch wasn't fixable and had to be removed
	BatchStatusPending      = "pending"       // Not yet built or in the process of being built
	BatchStatusStagingReady = "staging_ready" // Batch is built but not deployed to staging yet
	BatchStatusQCReady      = "qc_ready"      // Batch is on staging and ready for QC pass
	BatchStatusFailedQC     = "failed_qc"     // On staging, but QC failed it; it needs to be pulled and fixed
	BatchStatusPassedQC     = "passed_qc"     // On staging, passed QC; it needs to be pulled from staging and pushed live
	BatchStatusLive         = "live"          // Batch has gone live; batch and its issues need to be archived
	BatchStatusLiveDone     = "live_done"     // Batch has gone live; batch and its issues have been archived and are no longer on the filesystem
)

// BatchStatus describes the metadata corresponding to a database status
type BatchStatus struct {
	Status      string
	Live        bool
	Staging     bool
	Dead        bool
	NeedsAction bool
	Description string
}

var noStatus BatchStatus

var statusMap = map[string]BatchStatus{
	BatchStatusPending: {
		Status:      BatchStatusPending,
		Live:        false,
		Staging:     false,
		Dead:        false,
		NeedsAction: false,
		Description: "Pending: build job is scheduled but hasn't yet run"},
	BatchStatusStagingReady: {
		Status:      BatchStatusStagingReady,
		Live:        false,
		Staging:     false,
		Dead:        false,
		NeedsAction: true,
		Description: "Ready for ingest onto staging server"},
	BatchStatusQCReady: {
		Status:      BatchStatusQCReady,
		Live:        false,
		Staging:     true,
		Dead:        false,
		NeedsAction: true,
		Description: "On staging, awaiting quality control check"},
	BatchStatusFailedQC: {
		Status:      BatchStatusFailedQC,
		Live:        false,
		Staging:     true,
		Dead:        false,
		NeedsAction: true,
		Description: "Failed quality control, awaiting batch maintainer fixes"},
	BatchStatusDeleted: {
		Status:      BatchStatusDeleted,
		Live:        false,
		Staging:     false,
		Dead:        true,
		NeedsAction: false,
		Description: "Removed from the system.  Likely rebuilt under a new name."},
	BatchStatusPassedQC: {
		Status:      BatchStatusPassedQC,
		Live:        false,
		Staging:     true,
		Dead:        false,
		NeedsAction: true,
		Description: "Passed quality control, awaiting batch maintainer's push to production"},
	BatchStatusLive: {
		Status:      BatchStatusLive,
		Live:        true,
		Staging:     false,
		Dead:        false,
		NeedsAction: true,
		Description: "Live in production, awaiting archiving"},
	BatchStatusLiveDone: {
		Status:      BatchStatusLiveDone,
		Live:        true,
		Staging:     false,
		Dead:        false,
		NeedsAction: false,
		Description: "Live in production and archived: no longer available in NCA workflow"},
}

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
	StatusMeta  BatchStatus `sql:"-"`
	Location    string

	issues []*Issue
}

func bs(s string) BatchStatus {
	return statusMap[s]
}

// findBatches wraps all the job finding functionality so helpers can be
// one-liners.  This is purposely *not* exported to enforce a stricter API.
//
// NOTE: All instantiations from the database must go through this function to
// properly deserialize their data!
func findBatches(where string, args ...any) ([]*Batch, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var list []*Batch
	op.Select("batches", &Batch{}).Where(where, args...).AllObjects(&list)
	for _, j := range list {
		var err = j.deserialize()
		if err != nil {
			return nil, fmt.Errorf("error decoding batch %d: %s", j.ID, err)
		}
	}
	return list, op.Err()
}

// FindBatch looks for a batch by its id
func FindBatch(id int) (*Batch, error) {
	var list, err = findBatches("id = ?", id)
	if len(list) == 0 {
		return nil, err
	}
	return list[0], err
}

// InProcessBatches returns the full list of in-process batches (not live, not pending)
func InProcessBatches() ([]*Batch, error) {
	return findBatches("status IN (?, ?, ?, ?)", BatchStatusStagingReady, BatchStatusQCReady, BatchStatusFailedQC, BatchStatusPassedQC)
}

// FindLiveArchivedBatches returns all batches that are still live, but have an
// archived_at value
func FindLiveArchivedBatches() ([]*Batch, error) {
	return findBatches("status = ? AND archived_at > ?", BatchStatusLive, time.Time{})
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

	var chksum = crc32.ChecksumIEEE([]byte(webroot))
	b.Name = RandomBatchName(uint32(b.ID) + chksum)
	for _, i := range issues {
		i.BatchID = b.ID
		i.SaveOp(op, ActionTypeInternalProcess, SystemUser.ID, fmt.Sprintf("added to batch %q", b.Name))
	}

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

	var issues, err = Issues().BatchID(b.ID).Fetch()
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
		i.SaveOp(op, ActionTypeInternalProcess, SystemUser.ID, fmt.Sprintf("removed from batch %q - batch deleted", b.Name))
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

func (b *Batch) deserialize() error {
	b.StatusMeta = bs(b.Status)
	if b.StatusMeta == noStatus {
		return fmt.Errorf("invalid status %q on batch %s", b.Status, b.FullName())
	}

	return nil
}
