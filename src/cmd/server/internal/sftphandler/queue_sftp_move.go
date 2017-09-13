package sftphandler

import (
	"db"
	"jobs"
	"logger"
)

func queueSFTPIssueMove(i *Issue) (ok bool, status string) {
	// Find a DB issue or create one
	var dbi, err = db.FindIssueByKey(i.Key())
	if err != nil {
		logger.Critical("Unable to search for database issue %q: %s", i.Key(), err)
		return false, "Unable to connect to the database.  Try again or contact the system administrator."
	}

	if dbi == nil {
		dbi, err = createDatabaseIssue(i)
		if err != nil {
			logger.Critical("Unable to save a new database issue: %s", err)
			return false, "Unable to connect to the database.  Try again or contact the system administrator."
		}
	}

	// Look for an existing job for this issue.  If anything exists, we have to
	// make sure they're all failed move jobs.  We're okay closing and retrying a
	// failed move, but anything else is a problem.

	// All's well - queue up the job
	err = jobs.QueueSFTPIssueMove(dbi, i.Location)
	if err != nil {
		logger.Critical("Unable to queue issue %q for sftp move: %s", i.Key(), err)
		return false, "Error trying to queue issue.  Try again or contact the system administrator."
	}

	return true, "Issue queued successfully"
}

func createDatabaseIssue(i *Issue) (*db.Issue, error) {
	var dbi = &db.Issue{
		MARCOrgCode: conf.PDFBatchMARCOrgCode,
		LCCN:        i.Title.LCCN,
		Date:        i.DateStringReadable(),
		Edition:     i.Edition,
	}
	return dbi, dbi.Save()
}
