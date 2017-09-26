package jobs

import (
	"db"
	"logger"
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
