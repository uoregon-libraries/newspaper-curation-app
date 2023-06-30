package models

import (
	"fmt"
	"hash/crc32"
	"strings"
	"time"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// These are all possible batch status values
const (
	BatchStatusDeleted      = "deleted"       // Batch wasn't fixable and had to be removed
	BatchStatusPending      = "pending"       // Not yet built or in the process of being built
	BatchStatusStagingReady = "staging_ready" // Batch is built but not deployed to staging yet
	BatchStatusQCReady      = "qc_ready"      // Batch is on staging and ready for QC pass
	BatchStatusQCFlagIssues = "qc_flagging"   // Batch failed QC; problem issues need to be identified and removed
	BatchStatusPassedQC     = "passed_qc"     // On staging, passed QC; it needs to be pulled from staging and pushed live
	BatchStatusLive         = "live"          // Batch has gone live; batch and its issues need to be archived
	BatchStatusLiveArchived = "live_archived" // Batch is archived; its issues can be cleaned up in a few weeks
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
		Description: "Pending: build job is scheduled but hasn't yet run",
	},
	BatchStatusStagingReady: {
		Status:      BatchStatusStagingReady,
		Live:        false,
		Staging:     false,
		Dead:        false,
		NeedsAction: true,
		Description: "Ready for ingest onto staging server",
	},
	BatchStatusQCReady: {
		Status:      BatchStatusQCReady,
		Live:        false,
		Staging:     true,
		Dead:        false,
		NeedsAction: true,
		Description: "On staging, awaiting quality control check",
	},
	BatchStatusQCFlagIssues: {
		Status:      BatchStatusQCFlagIssues,
		Live:        false,
		Staging:     true,
		Dead:        false,
		NeedsAction: true,
		Description: "Failed quality control, awaiting QC issue flagging",
	},
	BatchStatusDeleted: {
		Status:      BatchStatusDeleted,
		Live:        false,
		Staging:     false,
		Dead:        true,
		NeedsAction: false,
		Description: "Removed from the system.  Likely rebuilt under a new name.",
	},
	BatchStatusPassedQC: {
		Status:      BatchStatusPassedQC,
		Live:        false,
		Staging:     true,
		Dead:        false,
		NeedsAction: true,
		Description: "Passed quality control, awaiting batch maintainer to load on production",
	},
	BatchStatusLive: {
		Status:      BatchStatusLive,
		Live:        true,
		Staging:     false,
		Dead:        false,
		NeedsAction: true,
		Description: "Live in production, awaiting archiving",
	},
	BatchStatusLiveArchived: {
		Status:      BatchStatusLiveArchived,
		Live:        true,
		Staging:     false,
		Dead:        false,
		NeedsAction: false,
		Description: "Live in production and archived: awaiting local file cleanup",
	},
	BatchStatusLiveDone: {
		Status:      BatchStatusLiveDone,
		Live:        true,
		Staging:     false,
		Dead:        false,
		NeedsAction: false,
		Description: "Live in production and archived: no longer available in NCA workflow",
	},
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
	WentLiveAt  time.Time
	Status      string
	StatusMeta  BatchStatus `sql:"-"`
	Location    string

	// NeedStagingPurge is true if the batch needs to be removed from staging;
	// this is just a stop-gap measure until more automation exists, at which
	// point this should be replaced
	NeedStagingPurge bool

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
			return nil, fmt.Errorf("error decoding batch %d: %w", j.ID, err)
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
	var statusList []any
	var placeholders []string
	for status, data := range statusMap {
		if data.NeedsAction {
			statusList = append(statusList, status)
			placeholders = append(placeholders, "?")
		}
	}
	var qry = fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ", "))
	return findBatches(qry, statusList...)
}

// FindLiveArchivedBatches returns all batches that are live and archived
func FindLiveArchivedBatches() ([]*Batch, error) {
	return findBatches("status = ?", BatchStatusLiveArchived)
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
		_ = i.SaveOp(op, ActionTypeInternalProcess, SystemUser.ID, fmt.Sprintf("added to batch %q", b.Name))
	}

	err = b.SaveOp(op)
	return b, err
}

// Job returns a new Job instance for manipulating this batch in some way
func (b *Batch) Job(t JobType, args map[string]string) *Job {
	var j = NewJob(t, args)
	j.ObjectID = b.ID
	j.ObjectType = JobObjectTypeBatch
	return j
}

// Issues pulls all issues from the database which have this batch's ID
func (b *Batch) Issues() ([]*Issue, error) {
	if len(b.issues) > 0 {
		return b.issues, nil
	}

	if b.ID == 0 {
		return b.issues, nil
	}

	var finder = Issues().BatchID(b.ID)
	if bs(b.Status).Live {
		finder = finder.AllowIgnored()
	}
	var issues, err = finder.Fetch()
	b.issues = issues
	return b.issues, err
}

// FlaggedIssues returns all issues flagged for removal from this batch
func (b *Batch) FlaggedIssues() ([]*FlaggedIssue, error) {
	return findFlaggedIssues("batch_id = ?", b.ID)
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
	// Validate the batch status before doing anything else
	var st = bs(b.Status)
	if st.Description == "" {
		return fmt.Errorf("invalid batch status: %s", b.Status)
	}
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

// FlagIssue marks an issue as needing to be removed from this batch
func (b *Batch) FlagIssue(i *Issue, who *User, reason string) error {
	// Caller should have already validated the batch and the issue, but we
	// *really* don't want data to be busted
	if b.Status != BatchStatusQCFlagIssues {
		return fmt.Errorf("cannot flag issue %s: batch %s is not allowed to have issues flagged", i.Key(), b.Name)
	}
	if i.BatchID != b.ID {
		return fmt.Errorf("cannot flag issue %s: not part of batch %s", i.Key(), b.Name)
	}

	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Exec(`INSERT INTO batches_flagged_issues (flagged_by_user_id, batch_id, issue_id, reason) VALUES (?, ?, ?, ?)`,
		who.ID, b.ID, i.ID, reason)
	return op.Err()
}

// UnflagIssue removes an issue from the "bad issue" queue
func (b *Batch) UnflagIssue(i *Issue) error {
	// Caller should validate this stuff, but just in case, we do it, too
	if b.Status != BatchStatusQCFlagIssues {
		return fmt.Errorf("cannot unflag issue %s: batch %s is not allowed to have issues flagged", i.Key(), b.Name)
	}
	if i.BatchID != b.ID {
		return fmt.Errorf("cannot unflag issue %s: not part of batch %s", i.Key(), b.Name)
	}

	// Technically we could have an "error" here if the issue wasn't flagged to
	// begin with. But in that case, alerting the user is potentially confusing
	// since the thing they want to happen is essentially already done.
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Exec(`DELETE FROM batches_flagged_issues WHERE batch_id = ? AND issue_id = ?`, b.ID, i.ID)
	return op.Err()
}

// AbortIssueFlagging returns the batch state to BatchStatusQCReady and removes
// all flagged issues tied to it
func (b *Batch) AbortIssueFlagging() error {
	if b.Status != BatchStatusQCFlagIssues && b.Status != BatchStatusPending {
		return fmt.Errorf("abort issue flagging: invalid batch status %s", b.Status)
	}

	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	b.Status = BatchStatusQCReady
	var err = b.SaveOp(op)
	if err != nil {
		return err
	}

	op.Exec(`DELETE FROM batches_flagged_issues WHERE batch_id = ?`, b.ID)
	return op.Err()
}

// EmptyFlaggedIssuesList clears issues flagged for removal from this batch
func (b *Batch) EmptyFlaggedIssuesList() error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug

	op.Exec(`DELETE FROM batches_flagged_issues WHERE batch_id = ?`, b.ID)
	return op.Err()
}

// SetLive flags a batch as being live as of now, and adjusts all its issues to be ignored by NCA
func (b *Batch) SetLive() error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	b.Status = BatchStatusLive
	b.WentLiveAt = time.Now()
	_ = b.SaveOp(op)
	op.Exec(`UPDATE issues SET ignored=1, workflow_step = ? WHERE batch_id = ?`, schema.WSInProduction, b.ID)

	return op.Err()
}

// Delete removes all issues from this batch and sets its status to "deleted".
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
		_ = i.SaveOp(op, ActionTypeInternalProcess, SystemUser.ID, fmt.Sprintf("removed from batch %q - batch deleted", b.Name))
	}
	return op.Err()
}

// Close finalizes a batch that's live and archived by setting its status to
// BatchStatusLiveDone.  This has some of our "safety first" business logic you
// don't get if you close the batch manually, e.g., it must be in the
// "live_archived" status and it must have been archived four weeks ago.
func (b *Batch) Close() error {
	var reqStatus = BatchStatusLiveArchived
	if b.Status != reqStatus {
		return fmt.Errorf("cannot close batch unless its status is %q", reqStatus)
	}
	var fourWeeksAgo = time.Now().Add(-time.Hour * 24 * 7 * 4)
	if !b.ArchivedAt.Before(fourWeeksAgo) {
		return fmt.Errorf("cannot close batches archived fewer than four weeks ago")
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
