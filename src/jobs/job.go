package jobs

import (
	"config"
	"db"
	"fmt"
	"logger"
	"os"
	"path/filepath"
	"schema"
	"strings"
)

// A Processor is a general interface for all database-driven jobs that process something
type Processor interface {
	Process(*config.Config) bool
	SetProcessSuccess(bool)
	JobID() int
	JobType() JobType
}

// Job wraps the DB job data and provides business logic for things like
// logging to the database
type Job struct {
	*db.Job
	Logger *logger.Logger
}

// JobID gets the underlying database job's id
func (j *Job) JobID() int {
	return j.Job.ID
}

// JobType converts the underlying database job's type to a proper JobType variable
func (j *Job) JobType() JobType {
	return JobType(j.Job.Type)
}

// SetProcessSuccess changes the process status to successful or failed and
// stores it, logging a critical error if the database operation fails
func (j *Job) SetProcessSuccess(success bool) {
	switch success {
	case true:
		j.Status = string(JobStatusSuccessful)
	case false:
		j.Status = string(JobStatusFailed)
	}
	var err = j.Save()
	if err != nil {
		j.Logger.Critical("Unable to update job status after completion (job: %d; success: %q): %s", j.ID, err)
	}
}

// Requeue closes out this job and queues a new, duplicate job
func (j *Job) Requeue() error {
	var op = db.DB.Operation()
	op.BeginTransaction()

	var clone = &db.Job{
		Type:     j.Type,
		ObjectID: j.ObjectID,
		Location: j.Location,
		Status:   string(JobStatusPending),
	}
	clone.Save()

	j.Status = string(JobStatusDone)
	j.Save()

	op.EndTransaction()
	return op.Err()
}

// IssueJob wraps the Job type to add things needed in all jobs tied to
// specific issues
type IssueJob struct {
	*Job
	Issue   *schema.Issue
	DBIssue *db.Issue
}

// jobLogWriter is our internal structure, which implements io.Writer in order
// to write to the database
type jobLogWriter struct {
	Job *Job
}

// Write implements io.Writer, splitting the logger output to produce log level
// and message strings for the database
func (jlw jobLogWriter) Write(msg []byte) (n int, err error) {
	// Kill trailing space, and turn newlines into literal \n so we can see them
	// if they are in any messages
	var line = strings.TrimSpace(string(msg))
	line = strings.Replace(line, "\n", "\\n", -1)

	// Duplicate the output to stderr so we have something to grep in cases where
	// looking at logs is easier
	fmt.Fprintf(os.Stderr, "%s (job id %d)\n", line, jlw.Job.ID)

	// Split the log message into its relevant parts
	var parts = strings.Split(line, " - ")
	if len(parts) < 4 {
		logger.Critical("Invalid logger message format")
		return 0, fmt.Errorf("invalid logger message format")
	}
	var level = parts[2]
	var message = parts[3]

	err = jlw.Job.WriteLog(level, message)
	if err != nil {
		logger.Critical("Unable to write log message: %s", err)
		return 0, err
	}

	return len(msg), nil
}

// NewJob wraps the given db.Job and sets up a logger
func NewJob(dbj *db.Job) *Job {
	var j = &Job{Job: dbj}
	j.Logger = &logger.Logger{
		TimeFormat: "2006/01/02 15:04:05.000",
		AppName:    filepath.Base(os.Args[0]),
		Output:     jobLogWriter{j},
	}
	return j
}
