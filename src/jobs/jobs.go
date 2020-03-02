package jobs

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

// DBJobToProcessor creates the appropriate structure or structures to get a
// database job's processor set up
func DBJobToProcessor(dbJob *db.Job) Processor {
	switch db.JobType(dbJob.Type) {
	case db.JobTypeSetIssueWS:
		return &SetIssueWS{IssueJob: NewIssueJob(dbJob)}
	case db.JobTypeSetIssueMasterLoc:
		return &SetIssueMasterLoc{IssueJob: NewIssueJob(dbJob)}
	case db.JobTypeSetIssueLocation:
		return &SetIssueLocation{IssueJob: NewIssueJob(dbJob)}
	case db.JobTypePageSplit:
		return &PageSplit{IssueJob: NewIssueJob(dbJob)}
	case db.JobTypeMakeDerivatives:
		return &MakeDerivatives{IssueJob: NewIssueJob(dbJob)}
	case db.JobTypeBuildMETS:
		return &BuildMETS{IssueJob: NewIssueJob(dbJob)}
	case db.JobTypeArchiveMasterFiles:
		return &ArchiveMasterFiles{IssueJob: NewIssueJob(dbJob)}
	case db.JobTypeSetBatchStatus:
		return &SetBatchStatus{BatchJob: NewBatchJob(dbJob)}
	case db.JobTypeSetBatchLocation:
		return &SetBatchLocation{BatchJob: NewBatchJob(dbJob)}
	case db.JobTypeCreateBatchStructure:
		return &CreateBatchStructure{BatchJob: NewBatchJob(dbJob)}
	case db.JobTypeMakeBatchXML:
		return &MakeBatchXML{BatchJob: NewBatchJob(dbJob)}
	case db.JobTypeWriteBagitManifest:
		return &WriteBagitManifest{BatchJob: NewBatchJob(dbJob)}
	case db.JobTypeSyncDir:
		return &SyncDir{Job: NewJob(dbJob)}
	case db.JobTypeKillDir:
		return &KillDir{Job: NewJob(dbJob)}
	case db.JobTypeRenameDir:
		return &RenameDir{Job: NewJob(dbJob)}
	case db.JobTypeCleanFiles:
		return &CleanFiles{Job: NewJob(dbJob)}
	default:
		logger.Errorf("Unknown job type %q for job id %d", dbJob.Type, dbJob.ID)
	}

	dbJob.Status = string(db.JobStatusFailed)
	dbJob.Save()
	return nil
}

// FindAllFailedJobs returns a list of all jobs which failed; these are not
// wrapped into IssueJobs or Processors, as failed jobs aren't meant to be
// reprocessed (though they can be requeued by creating new jobs)
func FindAllFailedJobs() (jobs []*Job) {
	var dbJobs, err = db.FindJobsByStatus(db.JobStatusFailed)
	if err != nil {
		logger.Criticalf("Unable to look up failed jobs: %s", err)
		return
	}

	for _, dbj := range dbJobs {
		jobs = append(jobs, NewJob(dbj))
	}
	return
}
