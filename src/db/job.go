package db

import (
	"fmt"
	"strings"
	"time"

	"github.com/Nerdmaster/magicsql"
)

// Object types for consistently inserting into the database
const (
	JobObjectTypeBatch = "batch"
	JobObjectTypeIssue = "issue"
)

// JobType represents all possible jobs the system queues and processes
type JobType string

// The full list of job types
const (
	JobTypePageSplit                JobType = "page_split"
	JobTypeMoveIssueToWorkflow      JobType = "move_issue_to_workflow"
	JobTypeMoveIssueToPageReview    JobType = "move_issue_to_page_review"
	JobTypeMakeDerivatives          JobType = "make_derivatives"
	JobTypeBuildMETS                JobType = "build_mets"
	JobTypeMoveMasterFiles          JobType = "move_master_files"
	JobTypeCreateBatchStructure     JobType = "create_batch_structure"
	JobTypeMakeBatchXML             JobType = "make_batch_xml"
	JobTypeMoveBatchToReadyLocation JobType = "move_batch_to_ready_location"
	JobTypeWriteBagitManifest       JobType = "write_bagit_manifest"
)

// ValidJobTypes is the full list of job types which can exist in the jobs
// table, for use in validating command-line job queue processing
var ValidJobTypes = []JobType{
	JobTypePageSplit,
	JobTypeMoveIssueToWorkflow,
	JobTypeMoveIssueToPageReview,
	JobTypeMakeDerivatives,
	JobTypeBuildMETS,
	JobTypeMoveMasterFiles,
	JobTypeCreateBatchStructure,
	JobTypeMakeBatchXML,
	JobTypeMoveBatchToReadyLocation,
	JobTypeWriteBagitManifest,
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
	logs        []*JobLog

	// The job won't be run until sometime after RunAt; usually it's very close,
	// but the daemon doesn't pound the database every 5 milliseconds, so it can
	// take a little bit
	RunAt time.Time

	// Extra information any job might need - e.g., the issue's next workflow
	// step if the job is successful
	ExtraData string

	// QueueJobID tells us which job (if any) should be queued up after this one
	// completes successfully
	QueueJobID int
}

// FindJob gets a job by its id
func FindJob(id int) (*Job, error) {
	var jobs, err = findJobs("id = ?", id)
	if len(jobs) == 0 {
		return nil, err
	}
	return jobs[0], err
}

// findJobs wraps all the job finding functionality so helpers can be
// one-liners.  This is purposely *not* exported to enforce a stricter API.
func findJobs(where string, args ...interface{}) ([]*Job, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var list []*Job
	op.Select("jobs", &Job{}).Where(where, args...).AllObjects(&list)
	return list, op.Err()
}

// PopNextPendingJob is a helper for locking the database to pull the oldest
// job with one of the given types and set it to in-process
func PopNextPendingJob(types []JobType) (*Job, error) {
	var op = DB.Operation()
	op.Dbg = Debug

	op.BeginTransaction()
	defer op.EndTransaction()

	// Wrangle the IN pain...
	var j = &Job{}
	var args []interface{}
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

	j.Status = string(JobStatusInProcess)
	j.StartedAt = time.Now()
	j.SaveOp(op)

	return j, op.Err()
}

// FindJobsByStatus returns all jobs that have the given status
func FindJobsByStatus(st JobStatus) ([]*Job, error) {
	return findJobs("status = ?", st)
}

// FindJobsByStatusAndType returns all jobs of the given status and type
func FindJobsByStatusAndType(st JobStatus, t JobType) ([]*Job, error) {
	return findJobs("status = ? AND job_type = ?", st, t)
}

// FindRecentJobsByType grabs all jobs of the given type which were created
// within the given duration or are still pending, for use in pulling lists of
// issues which are in the process of doing something
func FindRecentJobsByType(t JobType, d time.Duration) ([]*Job, error) {
	var pendingJobs, otherJobs []*Job
	var err error

	pendingJobs, err = FindJobsByStatusAndType(JobStatusPending, t)
	if err != nil {
		return nil, err
	}
	otherJobs, err = findJobs("status <> ? AND job_type = ? AND created_at > ?", string(JobStatusPending), t, time.Now().Add(-d))
	if err != nil {
		return nil, err
	}

	return append(pendingJobs, otherJobs...), nil
}

// FindJobsForIssueID returns all jobs tied to the given issue
func FindJobsForIssueID(id int) ([]*Job, error) {
	return findJobs("object_id = ?", id)
}

// Logs lazy-loads all logs for this job from the database
func (j *Job) Logs() []*JobLog {
	if j.logs == nil {
		var op = DB.Operation()
		op.Dbg = Debug
		op.Select("job_logs", &JobLog{}).Where("job_id = ?", j.ID).AllObjects(&j.logs)
	}

	return j.logs
}

// WriteLog stores a log message on this job
func (j *Job) WriteLog(level string, message string) error {
	var l = &JobLog{JobID: j.ID, LogLevel: level, Message: message}
	var op = DB.Operation()
	op.Dbg = Debug
	op.Save("job_logs", l)
	return op.Err()
}

// Save creates or updates the Job in the jobs table
func (j *Job) Save() error {
	var op = DB.Operation()
	op.Dbg = Debug
	return j.SaveOp(op)
}

// SaveOp creates or updates the job in the jobs table using a custom operation
func (j *Job) SaveOp(op *magicsql.Operation) error {
	op.Save("jobs", j)
	return op.Err()
}
