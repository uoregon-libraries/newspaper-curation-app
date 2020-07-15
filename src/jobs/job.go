package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	ltype "github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// A Processor is a general interface for all database-driven jobs that process something
type Processor interface {
	// Process runs the job and returns whether it was successful
	Process(*config.Config) bool

	// Valid returns if the job can be processed.  This is separated from the
	// Process function because many jobs can just say "yes" in all cases while
	// others need to check that things like database records, which should
	// exist, actually do.  It's clearer to centralize validity checks.
	Valid() bool

	// MaxRetries tells the processor how many times this job may attempt to
	// re-run if it fails
	MaxRetries() int

	// DBJob returns the low-level database Job for updating status, etc.
	DBJob() *models.Job
}

// Job wraps the DB job data and provides business logic for things like
// logging to the database
type Job struct {
	db         *models.Job
	Logger     *ltype.Logger
	maxRetries int
}

// MaxRetries allows all jobs to implement the Processor interface without
// having to write this specific function
func (j *Job) MaxRetries() int {
	return j.maxRetries
}

// NewJob wraps the given models.Job and sets up a logger
func NewJob(dbj *models.Job) *Job {
	var j = &Job{db: dbj, maxRetries: 25}
	j.Logger = &ltype.Logger{Loggable: &jobLogger{Job: j, AppName: filepath.Base(os.Args[0])}}
	return j
}

// Find looks up the job in the database and wraps it
func Find(id int) *Job {
	var dbJob, err = models.FindJob(id)
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
func (j *Job) DBJob() *models.Job {
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
