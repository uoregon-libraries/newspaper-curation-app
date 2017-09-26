package jobs

import (
	"db"
	"logger"
	"schema"
)

// JobType represents all possible jobs the system queues and processes
type JobType string

// The full list of job types
const (
	JobTypePageSplit               JobType = "page_split"
	JobTypeSFTPIssueMove           JobType = "sftp_issue_move"
	JobTypeMoveIssueForDerivatives JobType = "move_issue_for_derivatives"
	JobTypeMakeDerivatives         JobType = "make_derivatives"
)

// JobStatus represents the different states in which a job can exist
type JobStatus string

// The full list of job statuses
const (
	JobStatusPending    JobStatus = "pending"     // Jobs needing to be processed
	JobStatusSuccessful JobStatus = "success"     // Jobs which were successful
	JobStatusFailed     JobStatus = "failed"      // Jobs which are complete, but did not succeed
	JobStatusFailedDone JobStatus = "failed_done" // Jobs we ignore - e.g., failed jobs which were rerun
)

// FindAllPendingJobs returns a list of all jobs needing processing
func FindAllPendingJobs() (processors []Processor) {
	var dbJobs, err = db.FindJobsByStatus(string(JobStatusPending))
	for _, ij := range issueJobFindWrapper(dbJobs, err, "find pending jobs") {
		switch JobType(ij.Type) {
		case JobTypeSFTPIssueMove:
			processors = append(processors, &SFTPIssueMover{IssueJob: ij})
		case JobTypePageSplit:
			processors = append(processors, &PageSplit{IssueJob: ij})
		case JobTypeMoveIssueForDerivatives:
			processors = append(processors, &MoveIssueForDerivatives{IssueJob: ij})
		case JobTypeMakeDerivatives:
			processors = append(processors, &MakeDerivatives{IssueJob: ij})
		default:
			logger.Error("Unknown job type %q for job id %d", ij.Type, ij.ID)
		}
	}

	return
}

// FindAllFailedJobs returns a list of all jobs which failed; these are not
// wrapped into IssueJobs or Processors, as failed jobs aren't meant to be
// reprocessed (though they can be requeued by creating new jobs)
func FindAllFailedJobs() (jobs []*Job) {
	var dbJobs, err = db.FindJobsByStatus(string(JobStatusFailed))
	if err != nil {
		logger.Critical("Unable to look up failed jobs: %s", err)
		return
	}

	for _, dbj := range dbJobs {
		jobs = append(jobs, NewJob(dbj))
	}
	return
}

// issueJobFindWrapper takes the response from most job-finding db functions
// and returns a list of IssueJobs, validating everything as needed and logging
// Critical errors when any DB operation failed
//
// TODO: Remove this and build a db Job converter than switches on the job type
// to determine exactly what needs to be created, then returns a Processor with
// all the information set up as needed.
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
	si, err = dbi.SchemaIssue()
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
