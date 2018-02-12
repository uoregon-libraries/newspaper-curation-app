package uploadedissuehandler

import (
	"db"
	"jobs"

	"github.com/uoregon-libraries/gopkg/logger"
)

func queueSFTPIssueMove(i *Issue) (ok bool, status string) {
	// Find a DB issue or create one
	var dbi, err = db.FindIssueByKey(i.Key())
	if err != nil {
		logger.Criticalf("Unable to search for database issue %q: %s", i.Key(), err)
		return false, "Unable to connect to the database.  Try again or contact the system administrator."
	}

	// If an issue exists, but isn't new, we have something broken going on
	if dbi != nil && dbi.Location != i.Location {
		logger.Errorf("Broken / duplicate issue %q: DB id %d; location %q", i.Key(), dbi.ID, dbi.Location)
		return false, "Broken / duplicate issue detected.  Double-check the issue date (this can cause false dupe flags) or else contact the system administrator."
	}

	if dbi == nil {
		dbi, err = createDatabaseIssue(i)
		if err != nil {
			logger.Criticalf("Unable to save a new database issue: %s", err)
			return false, "Unable to connect to the database.  Try again or contact the system administrator."
		}
	}

	// Look for an existing job for this issue.  If anything exists, we have to
	// make sure they're all failed move jobs.  We're okay closing and retrying a
	// failed move, but anything else is a problem.
	var jobList []*db.Job
	jobList, err = db.FindJobsForIssueID(dbi.ID)
	if err != nil {
		logger.Criticalf("Unable to query jobs associated with issue %q: %s", i.Key(), err)
		return false, "Error verifying issue status.  Try again or contact the system administrator."
	}
	for _, job := range jobList {
		switch jobs.JobStatus(job.Status) {
		case jobs.JobStatusFailed:
			// We can just assume the user wasn't worried about the job's message,
			// and re-queued anyway.  We're therefore okay with a silent (to the end
			// user) failure if we can't close the job out, because requeueing is
			// more important to keep things flowing.  But we MUST alert somebody
			// that the old job is "stuck" in a failed state, as that could get
			// really weird if we mass-retry failed jobs.
			job.Status = string(jobs.JobStatusFailedDone)
			err = job.Save()
			if err != nil {
				logger.Criticalf("Unable to close failed job!  Manually fix this!  Job id %d; error: %s", job.ID, err)
			}
		case jobs.JobStatusFailedDone:
			continue
		default:
			logger.Criticalf("Unexpected job detected for issue %q (db id %d): job id %d, status %q",
				i.Key(), dbi.ID, job.ID, job.Status)
			return false, "Previous / broken job detected.  Contact the system administrator for help."
		}
	}

	// All's well - queue up the job
	err = jobs.QueueSFTPIssueMove(dbi, i.Location)
	if err != nil {
		logger.Criticalf("Unable to queue issue %q for sftp move: %s", i.Key(), err)
		return false, "Error trying to queue issue.  Try again or contact the system administrator."
	}

	searcher.ForceReload()

	return true, "Issue queued successfully"
}

func createDatabaseIssue(i *Issue) (*db.Issue, error) {
	var dbi = db.NewIssue(conf.PDFBatchMARCOrgCode, i.Title.LCCN, i.DateStringReadable(), i.Edition)
	dbi.Location = i.Location
	return dbi, dbi.Save()
}
