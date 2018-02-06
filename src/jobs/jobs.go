package jobs

import (
	"db"
	"fmt"

	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
)

// JobType represents all possible jobs the system queues and processes
type JobType string

// The full list of job types
const (
	JobTypePageSplit               JobType = "page_split"
	JobTypeSFTPIssueMove           JobType = "sftp_issue_move"
	JobTypeMoveIssueForDerivatives JobType = "move_issue_for_derivatives"
	JobTypeMakeDerivatives         JobType = "make_derivatives"
	JobTypeBuildMETS               JobType = "build_mets"
)

// ValidJobTypes is the full list of job types which can exist in the jobs
// table, for use in validating command-line job queue processing
var ValidJobTypes = []JobType{
	JobTypePageSplit,
	JobTypeSFTPIssueMove,
	JobTypeMoveIssueForDerivatives,
	JobTypeMakeDerivatives,
	JobTypeBuildMETS,
}

// JobStatus represents the different states in which a job can exist
type JobStatus string

// The full list of job statuses
const (
	JobStatusPending    JobStatus = "pending"     // Jobs needing to be processed
	JobStatusInProcess  JobStatus = "in_process"  // Jobs which have been taken by a worker but aren't done
	JobStatusSuccessful JobStatus = "success"     // Jobs which were successful
	JobStatusFailed     JobStatus = "failed"      // Jobs which are complete, but did not succeed
	JobStatusFailedDone JobStatus = "failed_done" // Jobs we ignore - e.g., failed jobs which were rerun
)

// DBJobToProcessor creates the appropriate structure or structures to get a
// database job's processor set up
func DBJobToProcessor(dbJob *db.Job) Processor {
	switch JobType(dbJob.Type) {
	case JobTypeSFTPIssueMove:
		return &SFTPIssueMover{IssueJob: NewIssueJob(dbJob)}
	case JobTypePageSplit:
		return &PageSplit{IssueJob: NewIssueJob(dbJob)}
	case JobTypeMoveIssueForDerivatives:
		return &MoveIssueForDerivatives{IssueJob: NewIssueJob(dbJob)}
	case JobTypeMakeDerivatives:
		return &MakeDerivatives{IssueJob: NewIssueJob(dbJob)}
	case JobTypeBuildMETS:
		return &BuildMETS{IssueJob: NewIssueJob(dbJob)}
	default:
		logger.Errorf("Unknown job type %q for job id %d", dbJob.Type, dbJob.ID)
		return nil
	}
}

// NextJobProcessor gets the oldest job with any of the given job types, sets
// it as in-process, and returns its Processor
func NextJobProcessor(types []string) Processor {
	var dbJob, err = popFirstPendingJob(types)

	if err != nil {
		logger.Errorf("Unable to pull next pending job: %s", err)
		return nil
	}
	if dbJob == nil {
		return nil
	}

	return DBJobToProcessor(dbJob)
}

// popFirstPendingJob is a helper for locking the database to pull the next pending job of
// the given type and setting it as being in-process
func popFirstPendingJob(types []string) (*db.Job, error) {
	var op = db.DB.Operation()
	op.Dbg = db.Debug

	op.BeginTransaction()
	defer op.EndTransaction()

	// Wrangle the IN pain...
	var j = &db.Job{}
	var args []interface{}
	var placeholders []string
	args = append(args, string(JobStatusPending))
	for _, t := range types {
		args = append(args, t)
		placeholders = append(placeholders, "?")
	}

	var clause = fmt.Sprintf("status = ? AND job_type IN (%s)", strings.Join(placeholders, ","))
	if !op.Select("jobs", &db.Job{}).Where(clause, args...).Order("created_at").First(j) {
		return nil, op.Err()
	}

	j.Status = string(JobStatusInProcess)
	j.StartedAt = time.Now()
	j.Save()

	return j, op.Err()
}

// FindAllFailedJobs returns a list of all jobs which failed; these are not
// wrapped into IssueJobs or Processors, as failed jobs aren't meant to be
// reprocessed (though they can be requeued by creating new jobs)
func FindAllFailedJobs() (jobs []*Job) {
	var dbJobs, err = db.FindJobsByStatus(string(JobStatusFailed))
	if err != nil {
		logger.Criticalf("Unable to look up failed jobs: %s", err)
		return
	}

	for _, dbj := range dbJobs {
		jobs = append(jobs, NewJob(dbj))
	}
	return
}
