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
	JobStatusPending    JobStatus = "pending" // Jobs needing to be processed
	JobStatusSuccessful JobStatus = "success" // Jobs which were successful
	JobStatusFailed     JobStatus = "failed"  // Jobs which are complete, but did not succeed
	JobStatusDone       JobStatus = "done"    // Jobs we ignore - e.g., failed jobs which were manually remedied
)

// FindJobsForIssue looks for and returns all jobs which are tied to the given issue's id
//
// TODO: If we have another use of ObjectID someday, we should put an
// ObjectType field in or something so we can differentiate issue jobs from
// other jobs tied to tables
func FindJobsForIssue(dbi *db.Issue) []*IssueJob {
	var dbJobs, err = db.FindJobsForIssueID(dbi.ID)
	return issueJobFindWrapper(dbJobs, err, fmt.Sprintf("find jobs for issue id %d", dbi.ID))
}

// FindPendingPageSplitJobs returns PageSplits that need to be processed
func FindPendingPageSplitJobs() (jobs []*PageSplit) {
	var dbJobs, err = db.FindJobsByStatusAndType(string(JobStatusPending), string(JobTypePageSplit))
	for _, ij := range issueJobFindWrapper(dbJobs, err, "find issues needing page splitting") {
		jobs = append(jobs, &PageSplit{IssueJob: ij})
	}

	return
}

func FindPendingSFTPIssueMoverJobs() (jobs []*SFTPIssueMover) {
	var dbJobs, err = db.FindJobsByStatusAndType(string(JobStatusPending), string(JobTypeSFTPIssueMove))
	for _, ij := range issueJobFindWrapper(dbJobs, err, "find sftp issues needing to be moved") {
		jobs = append(jobs, &SFTPIssueMover{IssueJob: ij})
	}

	return
}

// issueJobFindWrapper takes the response from most job-finding db functions
// and returns a list of IssueJobs, validating everything as needed and logging
// Critical errors when any DB operation failed
func issueJobFindWrapper(dbJobs []*db.Job, err error, onErrorMessage string) (issueJobs []*IssueJob) {
	if err != nil {
		logger.Critical("Unable to %s: %s", onErrorMessage, err)
		return
	}

	for _, dbJob := range dbJobs {
		var j = dbJobToIssueJob(dbJob)
		if j == nil {
			continue
		}
		issueJobs = append(issueJobs, j)
	}
	return
}

// dbJobToIssueJob setups up an IssueJob from a database Job, centralizing the
// common validations and data manipulation
func dbJobToIssueJob(dbJob *db.Job) *IssueJob {
	var dbi, err = db.FindIssue(dbJob.ObjectID)
	if err != nil {
		logger.Critical("Unable to find issue for job %d: %s", dbJob.ID, err)
		return nil
	}

	var si *schema.Issue
	si, err = dbToSchemaIssue(dbi)
	if err != nil {
		logger.Critical("Unable to prepare a schema.Issue for database issue %d: %s", dbi.ID, err)
		return nil
	}

	return &IssueJob{
		Job:     NewJob(dbJob),
		DBIssue: dbi,
		Issue:   si,
	}
}

// dbToSchemaIssue is a simple helper to make a job-friendly schema.Issue out
// of a database Issue
func dbToSchemaIssue(dbi *db.Issue) (*schema.Issue, error) {
	var dt, err = time.Parse("2006-01-02", dbi.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid time format (%s) in database issue", dbi.Date)
	}

	db.LoadTitles()
	var t = db.LookupTitle(dbi.LCCN).SchemaTitle()
	if t == nil {
		return nil, fmt.Errorf("missing title for issue ID %d", dbi.ID)
	}
	var si = &schema.Issue{
		Date:    dt,
		Edition: dbi.Edition,
		Title:   t,
	}
	return si, nil
}
