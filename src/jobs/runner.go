package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// runnerLogger implements logger.Loggable for logging runner-level information
type runnerLogger struct {
	ID      int32
	AppName string
	level   logger.LogLevel
}

func (l *runnerLogger) Log(level logger.LogLevel, message string) {
	if level < l.level {
		return
	}

	var timeString = time.Now().Format(logger.TimeFormat)
	fmt.Fprintf(os.Stderr, "%s - %s - %s - [runner %d] %s\n",
		timeString, l.AppName, level.String(), l.ID, message)
}

var runnerID int32

// nextRunnerID atomically generates a unique id for a runner to use in logging
func nextRunnerID() int32 {
	return atomic.AddInt32(&runnerID, 1)
}

// A Runner is responsible for popping jobs from the database and running them.
// A Runner will have a specific list of JobTypes it watches, and will check at
// regular intervals for those types of jobs.
type Runner struct {
	config     *config.Config
	jobTypes   []models.JobType
	identifier int32
	isDone     int32
	logger     *logger.Logger
}

// TODO: Put runners in the database so we can attach runner-level logs to the
// runner rather than having to dig through system logs.  Also, having a runner
// tied to a job, and a "last ping" or something on the runner table would make
// it easier to know when a runner died and needs to have its jobs restarted.

// NewRunner creates a Runner set up to look for a given list of job types
func NewRunner(c *config.Config, logLevel logger.LogLevel, jobTypes ...models.JobType) *Runner {
	var rid = nextRunnerID()
	return &Runner{
		config:     c,
		jobTypes:   jobTypes,
		identifier: rid,
		logger:     &logger.Logger{Loggable: &runnerLogger{ID: rid, level: logLevel, AppName: filepath.Base(os.Args[0])}},
	}
}

func (r *Runner) done() bool {
	return atomic.LoadInt32(&r.isDone) == 1
}

// Watch tells the Runner to check for its assigned job types at the given
// interval.  The interval is a duration between runs, not a time at which the
// runner is guaranteed to fire off, in order to be sure long-running jobs with
// short intervals aren't competing for resources.
//
// This will run forever and would typically be put into a goroutine.
func (r *Runner) Watch(interval time.Duration) {
	r.logger.Infof("Watching %q", r.jobTypes)

	var nextAttempt time.Time
	for !r.done() {
		if time.Now().After(nextAttempt) {
			r.loopAvailableJobs()
			nextAttempt = time.Now().Add(interval)
		}

		// Try not to eat all the CPU
		time.Sleep(time.Second)
	}

	r.logger.Infof("Done watching jobs")
}

// loopAvailableJobs runs a single "loop" of a runner's jobs, gathering all jobs in a
// ready state and running them, and catching any crashes in the process.
func (r *Runner) loopAvailableJobs() {
	// Catch any job-running crashes and report them. Hopefully these are
	// rare-to-never, as process() will get called a *lot*, and a job that's
	// mid-processing can get stuck indefinitely until we add some kind of
	// job-crash-resume mechanic here.
	defer func() {
		if err := recover(); err != nil {
			var buf = make([]byte, 100000)
			buf = buf[:runtime.Stack(buf, false)]
			r.logger.Criticalf("jobs: panic running a job: %v\n%s", err, buf)
		}
	}()

	// Loop until there aren't any jobs left to process
	for r.ProcessNextPendingJob() {
		// If r.done() became true, we need to stop looping and let nature take
		// its course....
		if r.done() {
			break
		}
	}
}

// Stop signals this job to stop looping once the current job is done
func (r *Runner) Stop() {
	r.logger.Infof("Received STOP request; attempting to clean up")
	atomic.StoreInt32(&r.isDone, 1)
}

// ProcessNextPendingJob gets the oldest job this runner can process, sets its
// status to in-process, and processes it.  If no processor was found, the
// return is false and nothing happens.
func (r *Runner) ProcessNextPendingJob() bool {
	var dbJob, err = models.PopNextPendingJob(r.jobTypes)

	if err != nil {
		r.logger.Errorf("Unable to pull next pending job: %s", err)
		return false
	}
	if dbJob == nil {
		return false
	}

	var j = DBJobToProcessor(dbJob)
	if j == nil {
		return false
	}
	r.process(j)
	return true
}

func (r *Runner) process(pr Processor) {
	var dbj = pr.DBJob()

	// Invalid jobs shouldn't realistically exist, but database errors have
	// occasionally been known to happen and we don't want runners panicking
	if !pr.Valid() {
		r.handleFailure(pr)
		return
	}

	r.logger.Infof("Starting job id %d (%q)", dbj.ID, dbj.Type)
	var resp = pr.Process(r.config)
	switch resp {
	case PRSuccess:
		r.handleSuccess(pr)
	case PRFailure:
		r.handleFailure(pr)
	case PRTryLater:
		r.handleTryLater(pr)
	default:
		r.logger.Fatalf("Invalid return from job Process(): %#v (job: %d)", resp, dbj.ID)
	}
}

func (r *Runner) handleSuccess(pr Processor) {
	var dbj = pr.DBJob()
	var err = models.CompleteJob(dbj)
	if err != nil {
		r.logger.Criticalf("Unable to update job status after success (job: %d): %s", dbj.ID, err)
		return
	}

	r.logger.Infof("Finished job id %d - success", dbj.ID)
}

func (r *Runner) handleFailure(pr Processor) {
	var dbj = pr.DBJob()
	if dbj.RetryCount >= pr.MaxRetries() {
		r.handleCriticalFailure(pr)
		return
	}

	var retryJob, err = dbj.FailAndRetry()
	if err != nil {
		r.logger.Criticalf("Unable to requeue failed job (job: %d): %s", dbj.ID, err)
		return
	}
	r.logger.Warnf("Failed job %d: retrying via job %d at %s (try #%d)",
		dbj.ID, retryJob.ID, retryJob.RunAt, retryJob.RetryCount)
}

func (r *Runner) handleTryLater(pr Processor) {
	var dbj = pr.DBJob()
	var err = dbj.TryLater(time.Minute)
	if err != nil {
		r.logger.Criticalf("Unable to set job to try later (job: %d): %s", dbj.ID, err)
		return
	}
}

func (r *Runner) handleCriticalFailure(pr Processor) {
	var dbj = pr.DBJob()
	dbj.Status = string(models.JobStatusFailed)

	var err = dbj.Save()
	if err != nil {
		r.logger.Criticalf("Unable to update job status after failure (job: %d): %s", dbj.ID, err)
		return
	}
	r.logger.Infof("Job id %d **failed** (see job logs)", dbj.ID)
}
