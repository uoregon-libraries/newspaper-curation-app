package uploads

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

func dbErr() apperr.Error {
	return &schema.IssueError{
		Err:  "database connection failure",
		Msg:  fmt.Sprintf("Error queueing issue.  Try again or contact the system administrator."),
		Prop: true,
	}
}

func invalidErr() apperr.Error {
	return &schema.IssueError{
		Err:  "issue is not valid for queueing",
		Msg:  fmt.Sprintf("Issue is no longer valid and may have been changed since attempting to queue.  Try again or contact the system administrator."),
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
	if len(i.Errors) > 0 {
		// This should be rare, but it can happen during normal operation, so we
		// just log an info message in case more digging needs to happen
		logger.Infof("Issue %q isn't able to be queued in uploads.Issue.Queue(): %#v", i.Key(), i.Errors)
		return invalidErr()
	}

	// Find a DB issue or create one
	var dbi, err = db.FindIssueByKey(i.Key())
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
	var jobList []*db.Job
	jobList, err = db.FindJobsForIssueID(dbi.ID)
	if err != nil {
		logger.Criticalf("Unable to query jobs associated with issue %q: %s", i.Key(), err)
		return dbErr()
	}
	for _, job := range jobList {
		switch jobs.JobStatus(job.Status) {
		case jobs.JobStatusFailed:
			job.Status = string(jobs.JobStatusFailedDone)
			err = job.Save()
			if err != nil {
				logger.Criticalf("Unable to close failed job!  Manually fix this!  Job id %d; error: %s", job.ID, err)
				return dbErr()
			}
		case jobs.JobStatusFailedDone:
			continue
		default:
			logger.Criticalf("Unexpected job detected for issue %q (db id %d): job id %d, status %q",
				i.Key(), dbi.ID, job.ID, job.Status)
			return brokenJobErr()
		}
	}

	// All's well - queue up the job
	switch i.WorkflowStep {
	case schema.WSSFTP:
		err = jobs.QueueSFTPIssueMove(dbi)
	case schema.WSScan:
		err = jobs.QueueMoveIssueForDerivatives(dbi)
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

func (i *Issue) createDatabaseIssue() (*db.Issue, error) {
	var dbi = db.NewIssue(i.MARCOrgCode, i.Title.LCCN, i.RawDate, i.Edition)

	// SFTP issues (for now) don't get their MOC set, so we have to do that here
	if dbi.MARCOrgCode == "" && i.WorkflowStep == schema.WSSFTP {
		dbi.MARCOrgCode = i.conf.PDFBatchMARCOrgCode
	}

	// Scanned issues need to be marked as such
	if i.WorkflowStep == schema.WSScan {
		dbi.IsFromScanner = true
	}

	dbi.Location = i.Location
	return dbi, dbi.Save()
}
