// metadata_jobs.go holds various very small jobs to update bits of metadata on
// issues and batches.  These mini-jobs allow us to do what was previously not
// an option: handle a failure to update statuses / states / location metadata.
// Jobs which aren't using this stuff yet will still have comments like "the
// job completed so all we can do here is loudly log failures".

package jobs

import (
	"strconv"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// SetIssueWS is a very simple job just to update the issue's workflow step in
// preparation for, or to reflect the conclusion of, another job
type SetIssueWS struct {
	*IssueJob
}

// Process updates the issue's workflow step and attempts to save it
func (j *SetIssueWS) Process(*config.Config) ProcessResponse {
	j.DBIssue.WorkflowStep = schema.WorkflowStep(j.db.Args[JobArgWorkflowStep])
	var err = j.DBIssue.SaveWithoutAction()
	if err != nil {
		j.Logger.Errorf("Unable to update workflow step for issue %d: %s", j.DBIssue.ID, err)
		return PRFailure
	}
	return PRSuccess
}

// SetIssueBackupLoc is another metadata job that just sets a single field for
// an issue: backup location
type SetIssueBackupLoc struct {
	*IssueJob
}

// Process updates the issue's backup location and attempts to save it
func (j *SetIssueBackupLoc) Process(*config.Config) ProcessResponse {
	j.DBIssue.BackupLocation = j.db.Args[JobArgLocation]
	var err = j.DBIssue.SaveWithoutAction()
	if err != nil {
		j.Logger.Errorf("Unable to update backup location for issue %d: %s", j.DBIssue.ID, err)
		return PRFailure
	}
	return PRSuccess
}

// SetIssueLocation just updates issues.location in the database
type SetIssueLocation struct {
	*IssueJob
}

// Process just updates the issue's location field
func (j *SetIssueLocation) Process(*config.Config) ProcessResponse {
	j.DBIssue.Location = j.db.Args[JobArgLocation]
	var err = j.DBIssue.SaveWithoutAction()
	if err != nil {
		j.Logger.Errorf("Error setting issue.location for id %d: %s", j.DBIssue.ID, err)
		return PRFailure
	}

	return PRSuccess
}

// SetIssueCurated sets the issue as having been curated by the given user. The
// issue logic is in its model; this  just allows us to execute that logic from
// our job runner.
type SetIssueCurated struct {
	*IssueJob
}

// Process validates the job id arg and sets the issue as having been curated
func (j *SetIssueCurated) Process(*config.Config) ProcessResponse {
	var arg = j.db.Args[JobArgID]
	var id, err = strconv.ParseInt(arg, 10, 64)
	if err != nil {
		j.Logger.Errorf("Error reading job arg (%q) as int64: %s; killing job", arg, err)
		return PRFatal
	}

	err = j.DBIssue.SetCurated(id)
	if err != nil {
		j.Logger.Errorf("Error setting issue %q (%d) as curated: %s", j.DBIssue.Key(), j.DBIssue.ID, err)
		return PRFailure
	}

	return PRSuccess
}

// IgnoreIssue sets an issue's "ignored" field to true
type IgnoreIssue struct {
	*IssueJob
}

// Process sets the ignored field to true and saves the issue
func (j *IgnoreIssue) Process(*config.Config) ProcessResponse {
	j.DBIssue.Ignored = true
	var err = j.DBIssue.SaveWithoutAction()
	if err != nil {
		j.Logger.Errorf("Error setting issue.ignored for id %d: %s", j.DBIssue.ID, err)
		return PRFailure
	}
	return PRSuccess
}

// PrepIssuePageLabels counts the JP2 files belonging to an issue and sets its
// page labels all to be blank
type PrepIssuePageLabels struct {
	*IssueJob
}

// Process reads and counts the JP2 files belonging to this issue, then fills
// in empty values for each page label
func (j *PrepIssuePageLabels) Process(*config.Config) ProcessResponse {
	j.DBIssue.PageLabels = make([]string, len(j.Issue.JP2Files()))
	var err = j.DBIssue.SaveWithoutAction()
	if err != nil {
		j.Logger.Errorf("Error storing empty pages for issue id %d: %s", j.DBIssue.ID, err)
		return PRFailure
	}
	return PRSuccess
}

// SetBatchStatus is another simple job which... wait for it... sets the status
// of the job's batch!
type SetBatchStatus struct {
	*BatchJob
}

// Process simply updates the batch status and saves to the database
func (j *SetBatchStatus) Process(*config.Config) ProcessResponse {
	j.DBBatch.Status = j.db.Args[JobArgBatchStatus]
	var err = j.DBBatch.SaveWithoutAction()
	if err != nil {
		j.Logger.Errorf("Unable to update status for batch %d: %s", j.DBBatch.ID, err)
		return PRFailure
	}
	return PRSuccess
}

// SetBatchLocation is a simple job to update a batch location after files are
// copied or movied somewhere
type SetBatchLocation struct {
	*BatchJob
}

// Process just updates the batch's location field
func (j *SetBatchLocation) Process(*config.Config) ProcessResponse {
	j.DBBatch.Location = j.db.Args[JobArgLocation]
	var err = j.DBBatch.SaveWithoutAction()
	if err != nil {
		j.Logger.Errorf("Error setting batch.location for id %d: %s", j.DBBatch.ID, err)
		return PRFailure
	}

	return PRSuccess
}

// RecordIssueAction adds an issue action to the Issue in question.  This one
// is slightly more involved than most metadata jobs, but in the end it's just
// a quick SQL INSERT, and an action, in my mind, really is just barely outside
// the traditional definition of metadata....
type RecordIssueAction struct {
	*IssueJob
}

// Process adds the issue action to the database
func (j *RecordIssueAction) Process(*config.Config) ProcessResponse {
	// This is a waste of cycles right here, but going through the Issue's save
	// procedure ensures that the action is created and associated with the issue
	// in a way that is consistent.  If we add things to how issues and actions
	// interact, we don't really want to duplicate (or else potentially break)
	// this consistency.  Oh... and I'm lazy.
	var err = j.DBIssue.Save(models.ActionTypeInternalProcess, models.SystemUser.ID, j.db.Args[JobArgMessage])
	if err != nil {
		j.Logger.Errorf("Error recording internal issue action for id %d: %s", j.DBIssue.ID, err)
		return PRFailure
	}

	return PRSuccess
}

// RecordBatchAction adds an action to the Batch in question
type RecordBatchAction struct {
	*BatchJob
}

// Process adds the action to the database
func (j *RecordBatchAction) Process(*config.Config) ProcessResponse {
	// Same as issue action: yes this is waste of cycles, but it ensures consistency
	var err = j.DBBatch.Save(models.ActionTypeInternalProcess, models.SystemUser.ID, j.db.Args[JobArgMessage])
	if err != nil {
		j.Logger.Errorf("Error recording internal batch action for id %d: %s", j.DBBatch.ID, err)
		return PRFailure
	}

	return PRSuccess
}

// MarkBatchLive sets a batch's go-live metadata and tells NCA to ignore all
// its issues, as they are no longer considered to be in the workflow
type MarkBatchLive struct {
	*BatchJob
}

// Process updates the batch status and go-live date, and saves to the database
func (j *MarkBatchLive) Process(*config.Config) ProcessResponse {
	var err = j.DBBatch.SetLive()
	if err != nil {
		j.Logger.Errorf("Unable to mark batch %d as being live: %s", j.DBBatch.ID, err)
		return PRFailure
	}
	return PRSuccess
}

// CancelJob simply ends a job that was either on hold or had failed by putting
// it into a finalized state (failed-done).
type CancelJob struct {
	*JobJob
}

// Process deals with changing the targeted job's status to failed_done while
// guarding against canceling jobs which should still stay as-is.
func (j *CancelJob) Process(*config.Config) ProcessResponse {
	var js = models.JobStatus(j.TargetJob.Status)
	if js != models.JobStatusOnHold && js != models.JobStatusFailed {
		j.Logger.Errorf("Cannot cancel job id %d: invalid job status (%q)", j.TargetJob.ID, j.TargetJob.Status)
		return PRFatal
	}

	j.TargetJob.Status = string(models.JobStatusFailedDone)
	var err = j.TargetJob.Save()
	if err != nil {
		j.Logger.Errorf("Cannot cancel job id %d: %s", j.TargetJob.ID, err)
		return PRFailure
	}

	return PRSuccess
}
