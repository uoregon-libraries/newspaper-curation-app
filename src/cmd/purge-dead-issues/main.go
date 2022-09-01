package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

const csi = "\033["
const ansiReset = csi + "0m"
const ansiBold = csi + "1m"
const ansiIntenseGreen = csi + "32;1m"
const ansiIntenseRed = csi + "31;1m"

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
	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}
	erroredIssuesPath = conf.ErroredIssuesPath
}

// issue is used for JSON output in the purged issues "report"
type issue struct {
	ID             int
	MARCOrgCode    string
	LCCN           string
	Date           string
	TitleName      string
	Location       string
	BackupLocation string
	IsFromScanner  bool
	Actions        []*models.Action
}

func main() {
	getConfig()

	logger.Debugf("Scanning for issues to purge")

	var issues, err = models.Issues().InWorkflowStep(schema.WSAwaitingProcessing).Fetch()
	if err != nil {
		logger.Fatalf("Unable to scan database for issues awaiting processing: %s", err)
	}

	// Set up a transaction so all jobs and issues are processed or skipped as
	// one operation
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()

	var purged []*models.Issue
	purged, err = purge(op, issues)

	if err == nil {
		writeReport(purged)
	}

	if err != nil {
		logger.Warnf("One or more errors were encountered: aborting transaction")
		op.Rollback()
	}

	if !opts.Live {
		logger.Warnf("DRY RUN: aborting transaction")
		op.Rollback()
	}

	op.EndTransaction()
	if op.Err() != nil {
		logger.Errorf("Failed finalizing purge-jobs' transaction: %s", err)
	}
	logger.Debugf("Process complete")
}

func writeReport(purged []*models.Issue) {
	// Print a "report"
	fmt.Printf("%s------------------------------------------------%s\n", ansiBold, ansiReset)
	if opts.Live {
		fmt.Printf("%s Purged Issue Report%s\n", ansiIntenseGreen, ansiReset)
	} else {
		fmt.Printf("%s DRY RUN%s: Purged Issue Report%s\n", ansiIntenseRed, ansiIntenseGreen, ansiReset)
	}
	fmt.Printf("%s (a JSON report is also written to purge.json)%s\n", ansiBold, ansiReset)
	fmt.Printf("%s------------------------------------------------%s\n", ansiBold, ansiReset)
	fmt.Println()
	var jsonIssues []*issue
	for _, i := range purged {
		jsonIssues = append(jsonIssues, &issue{
			ID:             i.ID,
			MARCOrgCode:    i.MARCOrgCode,
			LCCN:           i.LCCN,
			Date:           i.Date,
			TitleName:      i.Title.Name,
			Location:       i.Location,
			BackupLocation: i.BackupLocation,
			IsFromScanner:  i.IsFromScanner,
			Actions:        i.AllWorkflowActions(),
		})

		fmt.Printf("Issue %d (key: %q) from title %q\n", i.ID, i.Key(), i.Title.Name)
	}
	// Errors in marshaling shouldn't be possible with the current Issue structure
	var data, err = json.MarshalIndent(jsonIssues, "", "\t")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile("purge.json", data, 0644)
	if err != nil {
		logger.Errorf("Unable to write to purge.json: %s", err)
	}
}

func purge(dbop *magicsql.Operation, list []*models.Issue) (purged []*models.Issue, err error) {
	for _, i := range list {
		logger.Debugf("Examining issue id %d (%s)", i.ID, i.HumanName)
		var err = purgeIssue(dbop, i)
		if err == nil {
			logger.Infof("Issue id %d (%s): purged", i.ID, i.HumanName)
			purged = append(purged, i)
			continue
		}

		if errors.As(err, &fatalError{}) {
			logger.Warnf("Issue id %d (%s): fatal error checking purgability: %s", i.ID, i.HumanName, err)
			return nil, err
		}

		logger.Infof("Issue id %d (%s): skipping: %s", i.ID, i.HumanName, err)
	}

	return purged, nil
}

func purgeIssue(dbop *magicsql.Operation, i *models.Issue) error {
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
	err = doPurge(dbop, i, purgeJobs)
	if err != nil {
		err = fatalError{err}
	}

	return err
}

// doPurge finalizes the dead/on-hold jobs' statuses and then kicks off a new
// job to remove the issue using the existing QueueRemoveErroredIssue task.
func doPurge(dbop *magicsql.Operation, issue *models.Issue, purgeJobs []*models.Job) error {
	var purgeReason = "Issue failed getting through workflow:\n"
	for _, j := range purgeJobs {
		logger.Debugf("Canceling job id %d (%s)", j.ID, j.Type)
		if j.Status == string(models.JobStatusFailed) {
			purgeReason += fmt.Sprintf("- Job %d (%s) failed too many times\n", j.ID, j.Type)
		}

		var err = cancelJob(dbop, j)
		if err != nil {
			return err
		}
	}

	issue.SaveOp(dbop, models.ActionTypeInternalProcess, models.SystemUser.ID, purgeReason)
	var joblist = jobs.GetJobsForRemoveErroredIssue(issue, erroredIssuesPath)
	var err = jobs.QueueSerialOp(dbop, joblist...)
	if err != nil {
		return fmt.Errorf("queueing jobs to purge issue %d: %s", issue.ID, err)
	}

	return nil
}

// cancelJob deals with changing a job's status to failed_done while
// guarding against accidental purge-jobs
func cancelJob(dbop *magicsql.Operation, job *models.Job) error {
	switch models.JobStatus(job.Status) {
	case models.JobStatusOnHold, models.JobStatusFailed:
		job.Status = string(models.JobStatusFailedDone)
		job.SaveOp(dbop)
		return nil

	default:
		return fmt.Errorf("invalid job status for job id %d: %q", job.ID, job.Status)
	}
}
