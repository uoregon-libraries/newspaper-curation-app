package jobs

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// DBJobToProcessor creates the appropriate structure or structures to get a
// database job's processor set up
func DBJobToProcessor(dbJob *models.Job) Processor {
	switch models.JobType(dbJob.Type) {
	case models.JobTypeSetIssueWS:
		return &SetIssueWS{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypeSetIssueBackupLoc:
		return &SetIssueBackupLoc{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypeSetIssueLocation:
		return &SetIssueLocation{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypeFinalizeBatchFlaggedIssue:
		return &FinalizeBatchFlaggedIssue{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypeEmptyBatchFlaggedIssuesList:
		return &EmptyBatchFlaggedIssuesList{BatchJob: NewBatchJob(dbJob)}
	case models.JobTypeIgnoreIssue:
		return &IgnoreIssue{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypePageSplit:
		return &PageSplit{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypeMakeDerivatives:
		return &MakeDerivatives{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypeMoveDerivatives:
		return &MoveDerivatives{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypeBuildMETS:
		return &BuildMETS{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypeArchiveBackups:
		return &ArchiveBackups{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypeSetBatchStatus:
		return &SetBatchStatus{BatchJob: NewBatchJob(dbJob)}
	case models.JobTypeSetBatchLocation:
		return &SetBatchLocation{BatchJob: NewBatchJob(dbJob)}
	case models.JobTypeCreateBatchStructure:
		return &CreateBatchStructure{BatchJob: NewBatchJob(dbJob)}
	case models.JobTypeMakeBatchXML:
		return &MakeBatchXML{BatchJob: NewBatchJob(dbJob)}
	case models.JobTypeWriteActionLog:
		return &WriteActionLog{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypeWriteBagitManifest:
		return &WriteBagitManifest{BatchJob: NewBatchJob(dbJob)}
	case models.JobTypeValidateTagManifest:
		return &ValidateTagManifest{BatchJob: NewBatchJob(dbJob)}
	case models.JobTypeSyncDir:
		return &SyncDir{Job: NewJob(dbJob)}
	case models.JobTypeKillDir:
		return &KillDir{Job: NewJob(dbJob)}
	case models.JobTypeRenameDir:
		return &RenameDir{Job: NewJob(dbJob)}
	case models.JobTypeCleanFiles:
		return &CleanFiles{Job: NewJob(dbJob)}
	case models.JobTypeRemoveFile:
		return &RemoveFile{Job: NewJob(dbJob)}
	case models.JobTypeRenumberPages:
		return &RenumberPages{IssueJob: NewIssueJob(dbJob)}
	case models.JobTypeIssueAction:
		return &RecordIssueAction{IssueJob: NewIssueJob(dbJob)}
	default:
		logger.Errorf("Unknown job type %q for job id %d", dbJob.Type, dbJob.ID)
	}

	dbJob.Status = string(models.JobStatusFailed)
	dbJob.Save()
	return nil
}

// FindAllFailedJobs returns a list of all jobs which failed; these are not
// wrapped into IssueJobs or Processors, as failed jobs aren't meant to be
// reprocessed (though they can be requeued by creating new jobs)
func FindAllFailedJobs() (jobs []*Job) {
	var dbJobs, err = models.FindJobsByStatus(models.JobStatusFailed)
	if err != nil {
		logger.Criticalf("Unable to look up failed jobs: %s", err)
		return
	}

	for _, dbj := range dbJobs {
		jobs = append(jobs, NewJob(dbj))
	}
	return
}
