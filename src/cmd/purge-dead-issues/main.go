package main

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// Command-line options
type _opts struct {
	cli.BaseOptions
	Live bool `long:"live" description:"Run the issue purge operation rather than just showing what would be done"`
}

// Stupid error wrapper to make it easy to know if an issue's inability to
// purge was something critical like a database failure or just an issue that
// didn't meet the criteria
type fatalError struct {
	error
}

var opts _opts
var database *sql.DB
var erroredIssuesPath string

func getConfig() {
	var c = cli.New(&opts)
	c.AppendUsage(`Deletes all "stuck" issues couldn't make it into NCA.  Issues must have the "AwaitingProcessing" workflow step and at least one dead job ("failed", not "failed_done") to be considered for purging.  They will not be purged if they are tied to a batch or have any pending jobs associated with them.  All issues' jobs will be finalized (set to "failed_done") or removed (those that are on hold waiting for the failed job / jobs).`)

	var conf = c.GetConf()
	var err = dbi.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}
	erroredIssuesPath = conf.ErroredIssuesPath
}

func main() {
	getConfig()

	logger.Debugf("Scanning for issues to purge")

	var issues, err = models.FindIssuesAwaitingProcessing()
	if err != nil {
		logger.Fatalf("Unable to scan database for issues awaiting processing: %s", err)
	}

	for _, i := range issues {
		logger.Debugf("Examining issue id %d (%s)", i.ID, i.HumanName)
		err = purgeIssue(i)
		if err == nil {
			logger.Infof("Issue id %d (%s): purged", i.ID, i.HumanName)
			continue
		}

		if errors.As(err, &fatalError{}) {
			logger.Warnf("Issue id %d (%s): fatal error checking purgability: %s", i.ID, i.HumanName, err)
		} else {
			logger.Infof("Issue id %d (%s): skipping: %s", i.ID, i.HumanName, err)
		}
	}

	logger.Debugf("Process complete")
}

func purgeIssue(i *models.Issue) error {
	if i.BatchID != 0 {
		return fmt.Errorf("issue is tied to a batch")
	}

	var joblist, err = models.FindJobsForIssueID(i.ID)
	if err != nil {
		return &fatalError{err}
	}

	var purgeJobs []*models.Job
	var hasFailedJob bool
	for _, j := range joblist {
		switch models.JobStatus(j.Status) {
		// Jobs on hold / dead need to be closed out; failed jobs specifically get
		// called out so we know the issue is actually a valid purge candidate
		// (must have at least one dead job)
		case models.JobStatusFailed:
			hasFailedJob = true
			purgeJobs = append(purgeJobs, j)
		case models.JobStatusOnHold:
			purgeJobs = append(purgeJobs, j)

		// These types have no effect on processing and don't need to be dealt with
		// no matter what else happens
		case models.JobStatusSuccessful, models.JobStatusFailedDone:
			continue

		// These job types are blockers for purging an issue, because something
		// hasn't finished yet
		case models.JobStatusPending, models.JobStatusInProcess:
			return fmt.Errorf("cannot purge issue with pending or in-process jobs")

		// Make sure we account for every possible job type
		default:
			return fatalError{fmt.Errorf("invalid job status for job %d: %q", j.ID, j.Status)}
		}
	}

	if !hasFailedJob {
		return fmt.Errorf("cannot purge issue with no dead jobs")
	}

	// Getting here means no errors, so we can kick off the purge.  All errors
	// from doPurge are fatal.
	return fatalError{doPurge(i, purgeJobs)}
}

// doPurge finalizes the dead/on-hold jobs' statuses and then kicks off a new
// job to remove the issue using the existing QueueRemoveErroredIssue task.
func doPurge(issue *models.Issue, purgeJobs []*models.Job) error {
	// Set up a transaction so all jobs and the issue are processed or skipped as
	// one operation
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	for _, j := range purgeJobs {
		logger.Debugf("Canceling job id %d (%s)", j.ID, j.Type)
		var err = cancelJob(op, j)
		if err != nil {
			op.Rollback()
			return err
		}
	}

	var joblist = jobs.GetJobsForRemoveErroredIssue(issue, erroredIssuesPath)
	var err = jobs.QueueSerialOp(op, joblist...)
	if err != nil {
		op.Rollback()
		return fmt.Errorf("queueing jobs to purge issue %d: %s", issue.ID, err)
	}

	if !opts.Live {
		logger.Warnf("DRY RUN: aborting transaction")
		op.Rollback()
		return nil
	}

	op.EndTransaction()
	if op.Err() != nil {
		return fmt.Errorf("finalizing purge-jobs' transaction for issue %d: %s", issue.ID, err)
	}

	return nil
}

// cancelJob deals with changing a job's status to failed_done while
// guarding against accidental purge-jobs
func cancelJob(op *magicsql.Operation, job *models.Job) error {
	switch models.JobStatus(job.Status) {
	case models.JobStatusOnHold, models.JobStatusFailed:
		job.Status = string(models.JobStatusFailedDone)
		job.SaveOp(op)
		return nil

	default:
		return fmt.Errorf("invalid job status for job id %d: %q", job.ID, job.Status)
	}
}
