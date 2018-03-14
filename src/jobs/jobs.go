package jobs

import (
	"db"

	"github.com/uoregon-libraries/gopkg/logger"
)

// JobType represents all possible jobs the system queues and processes
type JobType string

// The full list of job types
const (
	JobTypePageSplit                JobType = "page_split"
	JobTypeMoveIssueToWorkflow      JobType = "move_issue_to_workflow"
	JobTypeMoveIssueToPageReview    JobType = "move_issue_to_page_review"
	JobTypeMakeDerivatives          JobType = "make_derivatives"
	JobTypeBuildMETS                JobType = "build_mets"
	JobTypeCreateBatchStructure     JobType = "create_batch_structure"
	JobTypeMakeBatchXML             JobType = "make_batch_xml"
	JobTypeMoveBatchToReadyLocation JobType = "move_batch_to_ready_location"
	JobTypeWriteBagitManifest       JobType = "write_bagit_manifest"
)

// ValidJobTypes is the full list of job types which can exist in the jobs
// table, for use in validating command-line job queue processing
var ValidJobTypes = []JobType{
	JobTypePageSplit,
	JobTypeMoveIssueToWorkflow,
	JobTypeMoveIssueToPageReview,
	JobTypeMakeDerivatives,
	JobTypeBuildMETS,
	JobTypeCreateBatchStructure,
	JobTypeMakeBatchXML,
	JobTypeMoveBatchToReadyLocation,
	JobTypeWriteBagitManifest,
}

// JobStatus represents the different states in which a job can exist
type JobStatus string

// The full list of job statuses
const (
	JobStatusOnHold     JobStatus = "on_hold"     // Jobs waiting for another job to complete
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
	case JobTypeMoveIssueToWorkflow:
		return &WorkflowIssueMover{IssueJob: NewIssueJob(dbJob)}
	case JobTypeMoveIssueToPageReview:
		return &PageReviewIssueMover{IssueJob: NewIssueJob(dbJob)}
	case JobTypePageSplit:
		return &PageSplit{IssueJob: NewIssueJob(dbJob)}
	case JobTypeMakeDerivatives:
		return &MakeDerivatives{IssueJob: NewIssueJob(dbJob)}
	case JobTypeBuildMETS:
		return &BuildMETS{IssueJob: NewIssueJob(dbJob)}
	case JobTypeCreateBatchStructure:
		return &CreateBatchStructure{BatchJob: NewBatchJob(dbJob)}
	case JobTypeMakeBatchXML:
		return &MakeBatchXML{BatchJob: NewBatchJob(dbJob)}
	case JobTypeMoveBatchToReadyLocation:
		return &MoveBatchToReadyLocation{BatchJob: NewBatchJob(dbJob)}
	case JobTypeWriteBagitManifest:
		return &WriteBagitManifest{BatchJob: NewBatchJob(dbJob)}
	default:
		logger.Errorf("Unknown job type %q for job id %d", dbJob.Type, dbJob.ID)
	}

	dbJob.Status = string(JobStatusFailed)
	dbJob.Save()
	return nil
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
