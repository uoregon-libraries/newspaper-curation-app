package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	ltype "github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

// A Processor is a general interface for all database-driven jobs that process something
type Processor interface {
	// Process runs the job and returns whether it was successful
	Process(*config.Config) bool

	// DBJob returns the low-level database Job for updating status, etc.
	DBJob() *db.Job
}

// Job wraps the DB job data and provides business logic for things like
// logging to the database
type Job struct {
	db     *db.Job
	Logger *ltype.Logger
}

// NewJob wraps the given db.Job and sets up a logger
func NewJob(dbj *db.Job) *Job {
	var j = &Job{db: dbj}
	j.Logger = &ltype.Logger{Loggable: &jobLogger{Job: j, AppName: filepath.Base(os.Args[0])}}
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

// DBJob implements job.Processor, returning the low-level database structure
func (j *Job) DBJob() *db.Job {
	return j.db
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

// Requeue closes out this job and queues a new, duplicate job ready for
// processing.  We do this instead of just rerunning a job so that the job logs
// can be tied to a distinct instance of a job, making it easier to debug
// things like command-line failures for a particular run.
func (j *Job) Requeue() error {
	var op = db.DB.Operation()
	op.BeginTransaction()

	// This is a shallow clone, but that should be fine since it's only the
	// top-level data that gets serialized to the database
	var clone db.Job = *j.db
	clone.ID = 0
	clone.Status = string(db.JobStatusPending)
	clone.SaveOp(op)
	j.db.Status = string(db.JobStatusFailedDone)
	j.db.SaveOp(op)

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
func (l *jobLogger) Log(level ltype.LogLevel, message string) {
	var timeString = time.Now().Format(ltype.TimeFormat)
	var msg = fmt.Sprintf("%s - %s - %s - [job %s:%d] %s\n",
		timeString, l.AppName, level.String(), l.db.Type, l.db.ID, message)
	var _, err = os.Stderr.WriteString(msg)
	if err != nil {
		_, err = fmt.Printf("ERROR: unable to write log message %q to STDERR: %s", msg, err)
		if err != nil {
			// Granted we probably won't see this, either, but we're out of options here....
			panic("Unable to write to STDERR or STDOUT")
		}
	}

	err = l.db.WriteLog(level.String(), message)
	if err != nil {
		logger.Criticalf("Unable to write log message: %s", err)
		return
	}
}
