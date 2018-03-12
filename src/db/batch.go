package db

import (
	"fmt"
	"time"

	"github.com/Nerdmaster/magicsql"
)

// These are all possible batch status values
const (
	BatchStatusPending   = "pending"    // Not yet built or in the process of being built
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
	Status      string
	Location    string

	issues []*Issue
}

// FindBatch looks for a batch by its id
func FindBatch(id int) (*Batch, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var b = &Batch{}
	var ok = op.Select("batches", b).Where("id = ?", id).First(b)
	if !ok {
		return nil, op.Err()
	}
	return b, op.Err()
}

// CreateBatch creates a batch in the database, using its ID to generate a
// unique batch name, and associating the given list of issues.  This is
// inefficient, but it gets the job done.
func CreateBatch(moc string, issues []*Issue) (*Batch, error) {
	var op = DB.Operation()
	op.Dbg = Debug
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

	b.Name = RandomBatchName(b.ID)
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
	var op = DB.Operation()
	op.Dbg = Debug
	return b.SaveOp(op)
}

// SaveOp saves the batch to the batches table with a custom operation for
// easier transactions
func (b *Batch) SaveOp(op *magicsql.Operation) error {
	op.Save("batches", b)
	return op.Err()
}
