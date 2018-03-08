package jobs

import (
	"config"
	"db"
	"fmt"
	"os"
	"path/filepath"
	"schema"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
)

// A Processor is a general interface for all database-driven jobs that process something
type Processor interface {
	// Process runs the job and returns whether it was successful
	Process(*config.Config) bool

	// UpdateWorkflow does any job-specific workflow manipulation, such as
	// changing the job's underlying object.  Only called on success.
	UpdateWorkflow()

	// DBJob returns the low-level database Job for updating status, etc.
	DBJob() *db.Job

	// ObjectLocation returns the job's object location, such as the directory in
	// which an issue resides, for the runner to use when updating future jobs
	ObjectLocation() string
}

// Job wraps the DB job data and provides business logic for things like
// logging to the database
type Job struct {
	*db.Job
	Logger *logger.Logger
}

// NewJob wraps the given db.Job and sets up a logger
func NewJob(dbj *db.Job) *Job {
	var j = &Job{Job: dbj}
	j.Logger = &logger.Logger{Loggable: &jobLogger{Job: j, AppName: filepath.Base(os.Args[0])}}
	return j
}

// Find looks up the job in the database and wraps it
func Find(id int) *Job {
	var dbJob, err = db.FindJob(id)
	if err != nil {
		logger.Errorf("Unable to look up job id %d: %s", id, err)
		return nil
	}
	if dbJob == nil {
		return nil
	}
	return NewJob(dbJob)
}

// DBJob returns the database job
func (j *Job) DBJob() *db.Job {
	return j.Job
}

// RunWhileTrue simplifies the common operation processors deal with when
// running a bunch of related operations, where the first failure needs to end
// the process entirely
func RunWhileTrue(subProcessors ...func() bool) (ok bool) {
	for _, subProc := range subProcessors {
		if !subProc() {
			return false
		}
	}

	return true
}

// Requeue closes out this job and queues a new, duplicate job
func (j *Job) Requeue() error {
	var op = db.DB.Operation()
	op.BeginTransaction()

	var clone = &db.Job{
		Type:             j.Type,
		ObjectID:         j.ObjectID,
		Location:         j.Location,
		Status:           string(JobStatusPending),
		RunAt:            j.RunAt,
		NextWorkflowStep: j.NextWorkflowStep,
		QueueJobID:       j.QueueJobID,
	}

	clone.SaveOp(op)

	j.Status = string(JobStatusFailedDone)
	j.SaveOp(op)

	op.EndTransaction()
	return op.Err()
}

// jobLogger implements logger.Loggable to write to stderr and the database
type jobLogger struct {
	*Job
	AppName string
}

// Log writes the pertinent data to stderr and the database so we can
// immediately see logs if we're watching for them, or search later against a
// specific job id's logs
func (l *jobLogger) Log(level logger.LogLevel, message string) {
	var timeString = time.Now().Format(logger.TimeFormat)
	fmt.Fprintf(os.Stderr, "%s - %s - %s - [job %s:%d] %s\n",
		timeString, l.AppName, level.String(), l.Job.Type, l.Job.ID, message)
	var err = l.Job.WriteLog(level.String(), message)
	if err != nil {
		logger.Criticalf("Unable to write log message: %s", err)
		return
	}
}

// IssueJob wraps the Job type to add things needed in all jobs tied to
// specific issues
type IssueJob struct {
	*Job
	Issue            *schema.Issue
	DBIssue          *db.Issue
	updateWorkflowCB func()
}

// NewIssueJob setups up an IssueJob from a database Job, centralizing the
// common validations and data manipulation
func NewIssueJob(dbJob *db.Job) *IssueJob {
	var dbi, err = db.FindIssue(dbJob.ObjectID)
	if err != nil {
		logger.Criticalf("Unable to find issue for job %d: %s", dbJob.ID, err)
		return nil
	}

	var si *schema.Issue
	si, err = dbi.SchemaIssue()
	if err != nil {
		logger.Criticalf("Unable to prepare a schema.Issue for database issue %d: %s", dbi.ID, err)
		return nil
	}

	return &IssueJob{
		Job:     NewJob(dbJob),
		DBIssue: dbi,
		Issue:   si,
	}
}

// Subdir returns a subpath to the job issue's directory for consistent
// directory naming and single-level paths
func (ij *IssueJob) Subdir() string {
	if ij.DBIssue.HumanName == "" {
		ij.DBIssue.HumanName = fmt.Sprintf("%s-%s-%d",
			ij.Issue.Title.LCCN, ij.Issue.DateEdition(), ij.DBIssue.ID)
	}
	return ij.DBIssue.HumanName
}

// WIPDir returns a hidden name for a work-in-progress directory to allow
// processing / copying to occur in a way that won't mess up end users
func (ij *IssueJob) WIPDir() string {
	return ".wip-" + ij.Subdir()
}

// UpdateWorkflow sets the attached issue's WorkflowStep if the job has defined
// a NextWorkflowStep.  The optional updateWorkflowCB is called if defined, and
// then the issue job is saved.  At this point, however, the job is complete,
// so all we can do is loudly log failures.
func (ij *IssueJob) UpdateWorkflow() {
	var ws = schema.WorkflowStep(ij.NextWorkflowStep)
	if ws != schema.WSNil {
		ij.DBIssue.WorkflowStep = ws
	}
	if ij.updateWorkflowCB != nil {
		ij.updateWorkflowCB()
	}

	var err = ij.DBIssue.Save()
	if err != nil {
		ij.Logger.Criticalf("Unable to update issue (dbid %d) workflow post-job: %s", ij.DBIssue.ID, err)
	}
}

// ObjectLocation implements he Processor interface
func (ij *IssueJob) ObjectLocation() string {
	return ij.DBIssue.Location
}
