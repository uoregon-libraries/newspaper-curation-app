package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
)

// Object types for consistently inserting into the database
const (
	JobObjectTypeBatch = "batch"
	JobObjectTypeIssue = "issue"
	JobObjectTypeJob   = "job"
)

// JobType represents all possible jobs the system queues and processes
type JobType string

// The full list of job types
const (
	JobTypeSetIssueWS                  JobType = "set_issue_workflow_step"
	JobTypeSetIssueBackupLoc           JobType = "set_issue_original_backup_location"
	JobTypeSetIssueLocation            JobType = "set_issue_location"
	JobTypeFinalizeBatchFlaggedIssue   JobType = "finalize_batch_flagged_issue"
	JobTypeEmptyBatchFlaggedIssuesList JobType = "empty_batch_flagged_issues_list"
	JobTypeIgnoreIssue                 JobType = "ignore_issue"
	JobTypeSetBatchStatus              JobType = "set_batch_status"
	JobTypeSetBatchNeedsStagingPurge   JobType = "set_batch_needs_staging_purge"
	JobTypePageSplit                   JobType = "page_split"
	JobTypeMakeDerivatives             JobType = "make_derivatives"
	JobTypeMoveDerivatives             JobType = "move_derivatives"
	JobTypeBuildMETS                   JobType = "build_mets"
	JobTypeArchiveBackups              JobType = "archive_backups"
	JobTypeSetBatchLocation            JobType = "set_batch_location"
	JobTypeCreateBatchStructure        JobType = "create_batch_structure"
	JobTypeMakeBatchXML                JobType = "make_batch_xml"
	JobTypeWriteActionLog              JobType = "write_action_log"
	JobTypeWriteBagitManifest          JobType = "write_bagit_manifest"
	JobTypeValidateTagManifest         JobType = "validate_tagmanifest"
	JobTypeMarkBatchLive               JobType = "mark_batch_live"
	JobTypeDeleteBatch                 JobType = "delete_batch"
	JobTypeSyncDir                     JobType = "sync_directory"
	JobTypeKillDir                     JobType = "delete_directory"
	JobTypeRenameDir                   JobType = "rename_directory"
	JobTypeCleanFiles                  JobType = "clean_files"
	JobTypeRemoveFile                  JobType = "remove_file"
	JobTypeRenumberPages               JobType = "renumber_pages"
	JobTypeIssueAction                 JobType = "record_issue_action"
	JobTypeCancelJob                   JobType = "cancel_job"
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
	JobTypeSetBatchStatus,
	JobTypeSetBatchNeedsStagingPurge,
	JobTypePageSplit,
	JobTypeMakeDerivatives,
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
	JobTypeSyncDir,
	JobTypeKillDir,
	JobTypeRenameDir,
	JobTypeCleanFiles,
	JobTypeRemoveFile,
	JobTypeRenumberPages,
	JobTypeIssueAction,
	JobTypeCancelJob,
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
	ID        int `sql:",primary"`
	JobID     int
	CreatedAt time.Time `sql:",readonly"`
	LogLevel  string
	Message   string
}

// A Job is anything the app needs to process and track in the background
type Job struct {
	ID          int       `sql:",primary"`
	CreatedAt   time.Time `sql:",readonly"`
	StartedAt   time.Time `sql:",noinsert"`
	CompletedAt time.Time `sql:",noinsert"`
	Type        string    `sql:"job_type"`
	ObjectID    int
	ObjectType  string
	Status      string
	PipelineID  int
	Sequence    int
	RetryCount  int
	logs        []*JobLog

	// The job won't be run until sometime after RunAt; usually it's very close,
	// but the daemon doesn't pound the database every 5 milliseconds, so it can
	// take a little bit
	RunAt time.Time

	// XDat holds extra information, encoded as JSON, any job might need - e.g.,
	// the issue's next workflow step if the job is successful.  This shouldn't
	// be modified directly; use Args instead (which is why we've chosen such an
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
func FindJob(id int) (*Job, error) {
	var jobs, err = findJobs("id = ?", id)
	if len(jobs) == 0 {
		return nil, err
	}
	return jobs[0], err
}

// countJobs returns a count of jobs matching the given where clause and an
// error if any is returned
func countJobs(where string, args ...any) (uint64, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var n = op.Select("jobs", &Job{}).Where(where, args...).Count().RowCount()
	return n, op.Err()
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
	op.Select("jobs", &Job{}).Where(where, args...).Order("created_at").AllObjects(&list)
	for _, j := range list {
		var err = j.decodeXDat()
		if err != nil {
			return nil, fmt.Errorf("error decoding job %d: %w", j.ID, err)
		}
	}
	return list, op.Err()
}

// PopNextPendingJob is a helper for locking the database to pull the oldest
// eligible job (pending + no higher-priority jobs in the pipeline) with one of
// the given types and set it to in-process
func PopNextPendingJob(types []JobType) (*Job, error) {
	// Wrangle the IN pain...
	var args []any
	var placeholders []string
	args = append(args, string(JobStatusPending), time.Now())
	for _, t := range types {
		args = append(args, string(t))
		placeholders = append(placeholders, "?")
	}
	var clause = fmt.Sprintf("status = ? AND run_at <= ? AND job_type IN (%s)", strings.Join(placeholders, ","))

	var jobs, err = findJobs(clause, args...)
	if len(jobs) == 0 {
		return nil, err
	}

	// Find the first eligible job: we already know we have the right status and
	// job type, but we now have to be sure no jobs above it in the pipeline are
	// waiting. This means that all higher-priority jobs must be either
	// successful or failed-done: all other statuses indicate something that's
	// waiting for NCA or a dev ("failed" jobs are stuck until a dev intervenes).
	for _, j := range jobs {
		var n, err = countJobs("pipeline_id = ? AND sequence < ? AND status NOT IN (?, ?)",
			j.PipelineID, j.Sequence, JobStatusFailedDone, JobStatusSuccessful)
		if err != nil {
			return nil, err
		}
		if n == 0 {
			j.Status = string(JobStatusInProcess)
			j.StartedAt = time.Now()
			return j, j.Save()
		}
	}

	return nil, nil
}

// FindJobsByStatus returns all jobs that have the given status
func FindJobsByStatus(st JobStatus) ([]*Job, error) {
	return findJobs("status = ?", string(st))
}

// FindJobsForIssueID returns all jobs tied to the given issue
func FindJobsForIssueID(id int) ([]*Job, error) {
	return findJobs("object_type = ? AND object_id = ?", JobObjectTypeIssue, id)
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

// Job returns a new job to manipulate *this* job. Jobception? I think we need
// one more layer to achieve it, but we're getting pretty close.
func (j *Job) Job(t JobType, args map[string]string) *Job {
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

// Clone returns a shallow copy of the job with key data cleared (database id,
// for instance)
func (j *Job) Clone() *Job {
	var clone *Job
	var temp = *j
	clone = &temp
	clone.ID = 0
	return clone
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
