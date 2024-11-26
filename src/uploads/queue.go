package uploads

import (
	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

func dbErr() apperr.Error {
	return &schema.IssueError{
		Err:  "database connection failure",
		Msg:  "Error queueing issue.  Try again or contact the system administrator.",
		Prop: true,
	}
}

func invalidErr() apperr.Error {
	return &schema.IssueError{
		Err:  "issue is not valid for queueing",
		Msg:  "Issue is no longer valid and may have been changed since attempting to queue.  Try again or contact the system administrator.",
		Prop: true,
	}
}

func brokenJobErr() apperr.Error {
	return &schema.IssueError{
		Err:  "unexpected / broken job",
		Msg:  "Broken / duplicate job already created.  Try again or contact the system administrator.",
		Prop: true,
	}
}

func badStepErr() apperr.Error {
	return &schema.IssueError{
		Err:  "invalid workflow step for queueing",
		Msg:  "This issue appears to already have been queued.  Try again or contact the system administrator.",
		Prop: true,
	}
}

// Queue attempts to send the issue to the workflow by queueing up a move job
func (i *Issue) Queue() apperr.Error {
	// Make sure the issue is definitely valid
	i.ValidateAll()
	if i.Errors.Major().Len() > 0 {
		// This should be rare, but it can happen during normal operation, so we
		// just log an info message in case more digging needs to happen
		logger.Infof("Issue %q isn't able to be queued in uploads.Issue.Queue(): %#v", i.Key(), i.Errors)
		return invalidErr()
	}

	// Find a DB issue or create one
	var dbi, err = models.FindIssueByKey(i.Key())
	if err != nil {
		logger.Criticalf("Unable to search for database issue %q: %s", i.Key(), err)
		return dbErr()
	}

	if dbi == nil {
		dbi, err = i.createDatabaseIssue()
		if err != nil {
			logger.Criticalf("Unable to save a new database issue: %s", err)
			return dbErr()
		}
	}

	// Look for an existing job for this issue.  If anything exists, we have to
	// make sure they're all failed move jobs.  We're okay closing and retrying a
	// failed move, but anything else is a problem.
	var jobList []*models.Job
	jobList, err = dbi.Jobs()
	if err != nil {
		logger.Criticalf("Unable to query jobs associated with issue %q: %s", i.Key(), err)
		return dbErr()
	}
	for _, job := range jobList {
		switch models.JobStatus(job.Status) {
		case models.JobStatusFailed:
			job.Status = string(models.JobStatusFailedDone)
			err = job.Save()
			if err != nil {
				logger.Criticalf("Unable to close failed job!  Manually fix this!  Job id %d; error: %s", job.ID, err)
				return dbErr()
			}
		case models.JobStatusFailedDone:
			continue
		case models.JobStatusPending:
			logger.Infof("Pending job detected for issue %q (db id %d): job id %d. Not attempting to queue issue a second time.", i.Key(), dbi.ID, job.ID)
			return nil
		default:
			logger.Criticalf("Unexpected job detected for issue %q (db id %d): job id %d, status %q",
				i.Key(), dbi.ID, job.ID, job.Status)
			return brokenJobErr()
		}
	}

	// All's well - queue up the job
	switch i.WorkflowStep {
	case schema.WSSFTP:
		err = jobs.QueueSFTPIssueMove(dbi, i.conf)
	case schema.WSScan:
		err = jobs.QueueMoveIssueForDerivatives(dbi, i.conf.WorkflowPath)
	default:
		logger.Criticalf("Invalid issue %q: workflow step %q isn't allowed for issue move jobs", i.Key(), i.WorkflowStep)
		return badStepErr()
	}
	if err != nil {
		logger.Criticalf("Unable to queue issue %q for move: %s", i.Key(), err)
		return dbErr()
	}

	return nil
}

// createDatabaseIssue converts the local issue's information into a
// DB-friendly version and saves it.
func (i *Issue) createDatabaseIssue() (*models.Issue, error) {
	var moc = i.MARCOrgCode
	var scanned = false

	// Set up special data that depends on the upload source
	switch i.WorkflowStep {
	case schema.WSSFTP:
		if moc == "" {
			moc = i.conf.PDFBatchMARCOrgCode
		}
	case schema.WSScan:
		scanned = true
	}

	return models.CreateIssueFromUpload(moc, i.Title.LCCN, i.RawDate, i.Edition, i.Location, scanned)
}
