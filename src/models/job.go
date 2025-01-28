package models

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
)

// Object types for consistently inserting into the database
const (
	JobObjectTypeJob   = "job"
	JobObjectTypeIssue = "issue"
	JobObjectTypeBatch = "batch"
)

// JobType represents all possible jobs the system queues and processes
type JobType string

// The full list of job types
const (
	// Job jobs (there is only one for now...)
	JobTypeCancelJob JobType = "cancel_job"

	// Jobs that are directly tied to an issue
	JobTypeArchiveBackups            JobType = "archive_backups"
	JobTypeBuildMETS                 JobType = "build_mets"
	JobTypeIgnoreIssue               JobType = "ignore_issue"
	JobTypeIssueAction               JobType = "record_issue_action"
	JobTypeMakeDerivatives           JobType = "make_derivatives"
	JobTypePrepIssuePageLabels       JobType = "prep_issue_page_labels"
	JobTypeMoveDerivatives           JobType = "move_derivatives"
	JobTypePageSplit                 JobType = "page_split"
	JobTypeRenumberPages             JobType = "renumber_pages"
	JobTypeSetIssueBackupLoc         JobType = "set_issue_original_backup_location"
	JobTypeSetIssueCurated           JobType = "set_issue_curated"
	JobTypeSetIssueLocation          JobType = "set_issue_location"
	JobTypeSetIssueWS                JobType = "set_issue_workflow_step"
	JobTypeWriteActionLog            JobType = "write_action_log"
	JobTypeFinalizeBatchFlaggedIssue JobType = "finalize_batch_flagged_issue"

	// Jobs that are directly tied to a batch
	JobTypeBatchAction                 JobType = "record_batch_action"
	JobTypeCreateBatchStructure        JobType = "create_batch_structure"
	JobTypeDeleteBatch                 JobType = "delete_batch"
	JobTypeEmptyBatchFlaggedIssuesList JobType = "empty_batch_flagged_issues_list"
	JobTypeMakeBatchXML                JobType = "make_batch_xml"
	JobTypeMarkBatchLive               JobType = "mark_batch_live"
	JobTypeSetBatchLocation            JobType = "set_batch_location"
	JobTypeSetBatchStatus              JobType = "set_batch_status"
	JobTypeValidateTagManifest         JobType = "validate_tagmanifest"
	JobTypeWriteBagitManifest          JobType = "write_bagit_manifest"
	JobTypeONILoadBatch                JobType = "oni_load_batch"
	JobTypeONIPurgeBatch               JobType = "oni_purge_batch"

	// Fairly general-purpose jobs, which use only the job args, not an object id
	JobTypeCleanFiles      JobType = "clean_files"
	JobTypeKillDir         JobType = "delete_directory"
	JobTypeRemoveFile      JobType = "remove_file"
	JobTypeRenameDir       JobType = "rename_directory"
	JobTypeSyncRecursive   JobType = "sync_recursive"
	JobTypeVerifyRecursive JobType = "verify_recursive"
	JobTypeMakeManifest    JobType = "make_manifest"
	JobTypeONIWaitForJob   JobType = "oni_wait_for_job"
)

// ValidJobTypes is the full list of job types which can exist in the jobs
// table, for use in validating command-line job queue processing
var ValidJobTypes = []JobType{
	JobTypeSetIssueWS,
	JobTypeSetIssueBackupLoc,
	JobTypeSetIssueLocation,
	JobTypeFinalizeBatchFlaggedIssue,
	JobTypeEmptyBatchFlaggedIssuesList,
	JobTypeIgnoreIssue,
	JobTypeSetIssueCurated,
	JobTypeSetBatchStatus,
	JobTypePageSplit,
	JobTypeMakeDerivatives,
	JobTypePrepIssuePageLabels,
	JobTypeMoveDerivatives,
	JobTypeBuildMETS,
	JobTypeArchiveBackups,
	JobTypeSetBatchLocation,
	JobTypeCreateBatchStructure,
	JobTypeMakeBatchXML,
	JobTypeWriteActionLog,
	JobTypeWriteBagitManifest,
	JobTypeValidateTagManifest,
	JobTypeMarkBatchLive,
	JobTypeDeleteBatch,
	JobTypeSyncRecursive,
	JobTypeVerifyRecursive,
	JobTypeKillDir,
	JobTypeRenameDir,
	JobTypeCleanFiles,
	JobTypeRemoveFile,
	JobTypeRenumberPages,
	JobTypeIssueAction,
	JobTypeBatchAction,
	JobTypeCancelJob,
	JobTypeMakeManifest,
	JobTypeONILoadBatch,
	JobTypeONIPurgeBatch,
	JobTypeONIWaitForJob,
}

// JobStatus represents the different states in which a job can exist
type JobStatus string

// The full list of job statuses
const (
	JobStatusOnHold     JobStatus = "on_hold"     // Jobs waiting for another job to complete
	JobStatusPending    JobStatus = "pending"     // Jobs needing to be processed
	JobStatusInProcess  JobStatus = "in_process"  // Jobs which have been taken by a worker but aren't done
	JobStatusSuccessful JobStatus = "success"     // Jobs which were successful
	JobStatusFailed     JobStatus = "failed"      // Jobs which are complete, but did not succeed
	JobStatusFailedDone JobStatus = "failed_done" // Jobs we ignore - e.g., failed jobs which were rerun
)

// JobLog is a single log entry attached to a job
type JobLog struct {
	ID        int64 `sql:",primary"`
	JobID     int64
	CreatedAt time.Time `sql:",readonly"`
	LogLevel  string
	Message   string
}

// A Job is anything the app needs to process and track in the background
type Job struct {
	ID          int64     `sql:",primary"`
	CreatedAt   time.Time `sql:",readonly"`
	StartedAt   time.Time `sql:",noinsert"`
	CompletedAt time.Time `sql:",noinsert"`
	Type        string    `sql:"job_type"`
	ObjectID    int64
	ObjectType  string
	Status      string
	PipelineID  int64
	Sequence    int
	RetryCount  int
	EntwineID   int64
	logs        []*JobLog

	// The job won't be run until sometime after RunAt. Usually it's very close,
	// but the daemon doesn't pound the database every 5 milliseconds, so it can
	// take a little bit
	RunAt time.Time

	// XDat holds extra information, encoded as JSON, any job might need - e.g.,
	// the issue's next workflow step if the job is successful.  This shouldn't
	// be modified directly: use Args instead (which is why we've chosen such an
	// odd name for this field).
	XDat string `sql:"extra_data"`

	// Args contains the decoded values from XDat
	Args map[string]string `sql:"-"`
}

// NewJob sets up a job of the given type as a pending job that's ready to run
// right away
func NewJob(t JobType, args map[string]string) *Job {
	if args == nil {
		args = make(map[string]string)
	}
	return &Job{
		Type:   string(t),
		Status: string(JobStatusPending),
		RunAt:  time.Now(),
		Args:   args,
	}
}

// FindJob gets a job by its id
func FindJob(id int64) (*Job, error) {
	var jobs, err = findJobs("id = ?", id)
	if len(jobs) == 0 {
		return nil, err
	}
	return jobs[0], err
}

// findJobs wraps all the job finding functionality so helpers can be
// one-liners.  This is purposely *not* exported to enforce a stricter API.
//
// NOTE: All instantiations from the database must go through this function to
// properly set up their args map!
func findJobs(where string, args ...any) ([]*Job, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var list []*Job
	op.Select("jobs", &Job{}).Where(where, args...).AllObjects(&list)
	for _, j := range list {
		var err = j.decodeXDat()
		if err != nil {
			return nil, fmt.Errorf("error decoding job %d: %w", j.ID, err)
		}
	}
	return list, op.Err()
}

// PopNextPendingJob is a helper for locking the database to pull the oldest
// job with one of the given types and set it to in-process
func PopNextPendingJob(types []JobType) (*Job, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug

	op.BeginTransaction()
	defer op.EndTransaction()

	// Wrangle the IN pain...
	var j = &Job{}
	var args []any
	var placeholders []string
	args = append(args, string(JobStatusPending), time.Now())
	for _, t := range types {
		args = append(args, string(t))
		placeholders = append(placeholders, "?")
	}

	var clause = fmt.Sprintf("status = ? AND run_at <= ? AND job_type IN (%s)", strings.Join(placeholders, ","))
	if !op.Select("jobs", &Job{}).Where(clause, args...).Order("created_at").First(j) {
		return nil, op.Err()
	}

	var err = j.decodeXDat()
	if err != nil {
		return nil, fmt.Errorf("error decoding job %d: %w", j.ID, err)
	}
	j.Status = string(JobStatusInProcess)
	j.StartedAt = time.Now()
	_ = j.SaveOp(op)

	// Make sure the pipeline's start date has been set, or else set it now
	var p *Pipeline
	p, err = findPipeline(j.PipelineID)
	if err != nil {
		return j, err
	}
	if p.StartedAt.IsZero() {
		p.StartedAt = time.Now()
		_ = p.saveOp(op)
	}

	return j, op.Err()
}

// FindUnfinishedJobs returns all jobs that aren't "complete". i.e., jobs that
// weren't successful and haven't failed.
func FindUnfinishedJobs() ([]*Job, error) {
	return findJobs("status NOT IN (?, ?, ?)", JobStatusSuccessful, JobStatusFailed, JobStatusFailedDone)
}

// FindJobsByStatusAndType returns all jobs that have the given status and job type
func FindJobsByStatusAndType(status JobStatus, typ JobType) ([]*Job, error) {
	return findJobs("status = ? AND job_type = ?", status, typ)
}

// Logs lazy-loads all logs for this job from the database
func (j *Job) Logs() []*JobLog {
	if j.logs == nil {
		var op = dbi.DB.Operation()
		op.Dbg = dbi.Debug
		op.Select("job_logs", &JobLog{}).Where("job_id = ?", j.ID).AllObjects(&j.logs)
	}

	return j.logs
}

// BuildJob returns a new job to manipulate *this* job. Jobception? I think we
// need one more layer to achieve it, but we're getting pretty close.
func (j *Job) BuildJob(t JobType, args map[string]string) *Job {
	var j2 = NewJob(t, args)
	j2.ObjectID = j.ID
	j2.ObjectType = JobObjectTypeJob
	return j2
}

// WriteLog stores a log message on this job
func (j *Job) WriteLog(level string, message string) error {
	var l = &JobLog{JobID: j.ID, LogLevel: level, Message: message}
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Save("job_logs", l)
	return op.Err()
}

// decodeXDat attempts to parse XDat
func (j *Job) decodeXDat() error {
	// Special case 1: no extra data means we don't try to decode it
	if j.XDat == "" {
		return nil
	}

	// Special case 2: raw extra data - we hard-code whatever is in XDat as
	// being a "legacy" value so the app at least doesn't crash, and we could
	// convert the data if necessary.
	if j.XDat[0:3] != "v.2" {
		j.Args = make(map[string]string)
		j.Args["legacy"] = j.XDat
		return nil
	}

	return json.Unmarshal([]byte(j.XDat[3:]), &j.Args)
}

// encodeArgs turns our args map into JSON.  We ignore errors here because it's
// not actually possible for Go's built-in JSON encoder to fail when we're just
// encoding a map of string->string.
func (j *Job) encodeArgs() {
	if len(j.Args) == 0 {
		j.XDat = ""
		return
	}
	var b, _ = json.Marshal(j.Args)
	j.XDat = "v.2" + string(b)
}

// Save creates or updates the Job in the jobs table
func (j *Job) Save() error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	return j.SaveOp(op)
}

// SaveOp creates or updates the job in the jobs table using a custom operation
func (j *Job) SaveOp(op *magicsql.Operation) error {
	j.encodeArgs()
	op.Save("jobs", j)
	return op.Err()
}

func countJobsOp(op *magicsql.Operation, where string, args ...any) uint64 {
	var n = op.Select("jobs", &Job{}).Where(where, args...).Count().RowCount()
	return n
}

// CompleteJob updates the job's status and completion time, then saves the
// job. If there are no other jobs with the same sequence, the next sequence's
// jobs are set to pending. If there are no jobs remaining at all, the pipeline
// is flagged as being completed.
//
// Though this function only takes a Job as a parameter, it mucks around with
// other jobs as well as the job's Pipeline, so it doesn't feel right to make
// it a function of Job as opposed to a standalone function.
func CompleteJob(j *Job) error {
	// We need the job's pipeline - if we can't get this, the rest of the
	// function doesn't really matter
	var p, err = findPipeline(j.PipelineID)
	if err != nil {
		return err
	}

	// Start a transaction as we might be manipulating a lot of entities here
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	j.Status = string(JobStatusSuccessful)
	j.CompletedAt = time.Now()
	_ = j.SaveOp(op)

	// If there are any unfinished jobs at the same sequence number, we don't
	// need to do anything more
	var n = countJobsOp(op, "pipeline_id = ? AND sequence <= ? AND status not in (?, ?)", j.PipelineID, j.Sequence, JobStatusFailedDone, JobStatusSuccessful)
	if n > 0 {
		return op.Err()
	}

	// If there are no jobs left, the pipeline is done and we can close it
	n = countJobsOp(op, "pipeline_id = ? AND status not in (?, ?)", j.PipelineID, JobStatusFailedDone, JobStatusSuccessful)
	if n == 0 {
		p.CompletedAt = time.Now()
		return p.saveOp(op)
	}

	// There are jobs left, but they're on hold, so let's fix that.
	//
	// TODO: DON'T hard-code the Sequence + 1 here! If we ever want jobs that
	// have space between them, or we allow an individual job to be queued but
	// then removed for some reason, this breaks and debugging would be a pain.
	// Probably not likely, but it's very little work to find the next sequence
	// instead of just assuming.
	op.Exec("UPDATE jobs SET status = ? WHERE pipeline_id = ? AND sequence = ?", JobStatusPending, j.PipelineID, j.Sequence+1)
	return op.Err()
}

// Clone returns a shallow copy of the job with key data cleared (database id,
// for instance)
func (j *Job) Clone() *Job {
	var clone *Job
	var temp = *j
	clone = &temp
	clone.ID = 0
	return clone
}

// QueueSiblingJobs adds the given list of jobs to the reference job's pipeline
// at its same sequence so they'll be executed before whatever would be next in
// the pipeline.
func (j *Job) QueueSiblingJobs(list []*Job) error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	for _, sibling := range list {
		sibling.PipelineID = j.PipelineID
		sibling.Sequence = j.Sequence
		_ = sibling.SaveOp(op)
	}

	return op.Err()
}

// EntwineJobs "connects" the passed-in jobs so that on any failure, the list
// as a whole is requeued instead of justthe job which failed. This should only
// be used for jobs where the *group* is idempotent **or** resilience is so
// critical that idempotence is worth losing.
//
// Please NEVER use this for jobs that aren't in the same pipeline!
func EntwineJobs(list []*Job) {
	rand.Seed(time.Now().UnixNano())
	var n = rand.Int63()
	for _, j := range list {
		j.EntwineID = n
	}
}

// TryLater updates the job's status back to pending and sets its run-at to now
// plus the given delay
func (j *Job) TryLater(delay time.Duration) error {
	j.Status = string(JobStatusPending)
	j.RunAt = time.Now().Add(delay)
	return j.Save()
}

// FailAndRetry closes out j and queues a new, duplicate job ready for
// processing.  We do this instead of just rerunning a job so that the job logs
// can be tied to a distinct instance of a job, making it easier to debug
// things like command-line failures for a particular run.
func (j *Job) FailAndRetry() (*Job, error) {
	var op = dbi.DB.Operation()
	op.BeginTransaction()

	var clone = j.Clone()
	clone.Status = string(JobStatusPending)
	clone.RetryCount++

	// Calculate the delay - essentially exponential backoff but starting at ~30
	// seconds and capping at 24 hours
	var delay = time.Second << uint(clone.RetryCount+3)
	var maxDelay = time.Hour * 24
	if delay > maxDelay {
		delay = maxDelay
	}
	clone.RunAt = time.Now().Add(delay)
	_ = clone.SaveOp(op)

	j.Status = string(JobStatusFailedDone)
	_ = j.SaveOp(op)

	op.EndTransaction()
	return clone, op.Err()
}

// RenewDeadJob takes a failed (NOT failed_done) job and queues a new job as if
// it were being created for the first time, and is set to run immediately.
//
// This is used after manual intervention for a job that exhausted all retries.
func RenewDeadJob(j *Job) (*Job, error) {
	if j.Status != string(JobStatusFailed) {
		return nil, fmt.Errorf("cannot restart unfailed job")
	}

	var op = dbi.DB.Operation()
	op.BeginTransaction()

	var clone = j.Clone()
	clone.Status = string(JobStatusPending)
	clone.RetryCount = 0
	clone.RunAt = time.Now()
	_ = clone.SaveOp(op)

	j.Status = string(JobStatusFailedDone)
	_ = j.SaveOp(op)

	op.EndTransaction()
	return clone, op.Err()
}
