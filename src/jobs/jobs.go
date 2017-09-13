package jobs

import (
	"db"
	"fmt"
	"logger"
	"schema"
	"time"
)

// JobType represents all possible jobs the system queues and processes
type JobType string

// The full list of job types
const (
	JobTypePageSplit     JobType = "page_split"
	JobTypeSFTPIssueMove JobType = "sftp_issue_move"
)

// JobStatus represents the different states in which a job can exist
type JobStatus string

// The full list of job statuses
const (
	JobStatusPending    JobStatus = "pending"
	JobStatusSuccessful JobStatus = "success"
)

// FindPendingPageSplitJobs returns PageSplits that need to be processed
func FindPendingPageSplitJobs() []*PageSplit {
	var dbJobs, err = db.FindJobsByStatusAndType(string(JobStatusPending), string(JobTypePageSplit))
	if err != nil {
		logger.Critical("Unable to find issues needing page splitting: %s", err)
		return nil
	}

	var pageSplits []*PageSplit
	for _, dbJob := range dbJobs {
		var dbi *db.Issue
		dbi, err = db.FindIssue(dbJob.ObjectID)
		if err != nil {
			logger.Critical("Unable to find issue for job %d: %s", dbJob.ID, err)
			continue
		}

		var si *schema.Issue
		si, err = dbToSchemaIssue(dbi)
		if err != nil {
			logger.Critical("Unable to prepare a schema.Issue for database issue %d: %s", dbi.ID, err)
			continue
		}

		pageSplits = append(pageSplits, &PageSplit{
			IssueJob: &IssueJob{
				Job: NewJob(dbJob),
				DBIssue: dbi,
				Issue:   si,
			},
		})
	}
	return pageSplits
}

// dbToSchemaIssue is a simple helper to make a job-friendly schema.Issue out
// of a database Issue
func dbToSchemaIssue(dbi *db.Issue) (*schema.Issue, error) {
	var dt, err = time.Parse("2006-01-02", dbi.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid time format (%s) in database issue", dbi.Date)
	}

	var t = db.LookupTitle(dbi.LCCN).SchemaTitle()
	var si = &schema.Issue{
		Date:    dt,
		Edition: dbi.Edition,
		Title:   t,
	}
	return si, nil
}
