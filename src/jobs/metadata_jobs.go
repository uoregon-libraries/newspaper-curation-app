// metadata_jobs.go holds various very small jobs to update bits of metadata on
// issues and batches.  These mini-jobs allow us to do what was previously not
// an option: handle a failure to update statuses / states / location metadata.
// Jobs which aren't using this stuff yet will still have comments like "the
// job completed so all we can do here is loudly log failures".

package jobs

import (
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
func (j *SetIssueWS) Process(*config.Config) bool {
	j.DBIssue.WorkflowStep = schema.WorkflowStep(j.db.Args[wsArg])
	var err = j.DBIssue.SaveWithoutAction()
	if err != nil {
		j.Logger.Errorf("Unable to update workflow step for issue %d: %s", j.DBIssue.ID, err)
	}
	return err == nil
}

// SetIssueBackupLoc is another metadata job that just sets a single field for
// an issue: backup location
type SetIssueBackupLoc struct {
	*IssueJob
}

// Process updates the issue's backup location and attempts to save it
func (j *SetIssueBackupLoc) Process(*config.Config) bool {
	j.DBIssue.BackupLocation = j.db.Args[locArg]
	var err = j.DBIssue.SaveWithoutAction()
	if err != nil {
		j.Logger.Errorf("Unable to update backup location for issue %d: %s", j.DBIssue.ID, err)
	}
	return err == nil
}

// SetIssueLocation just updates issues.location in the database
type SetIssueLocation struct {
	*IssueJob
}

// Process just updates the issue's location field
func (j *SetIssueLocation) Process(*config.Config) bool {
	j.DBIssue.Location = j.db.Args[locArg]
	var err = j.DBIssue.SaveWithoutAction()
	if err != nil {
		j.Logger.Errorf("Error setting issue.location for id %d: %s", j.DBIssue.ID, err)
		return false
	}

	return true
}

// IgnoreIssue sets an issue's "ignored" field to true
type IgnoreIssue struct {
	*IssueJob
}

// Process sets the ignored field to true and saves the issue
func (j *IgnoreIssue) Process(*config.Config) bool {
	j.DBIssue.Ignored = true
	var err = j.DBIssue.SaveWithoutAction()
	if err != nil {
		j.Logger.Errorf("Error setting issue.ignored for id %d: %s", j.DBIssue.ID, err)
		return false
	}

	return true
}

// SetBatchStatus is another simple job which... wait for it... sets the status
// of the job's batch!
type SetBatchStatus struct {
	*BatchJob
}

// Process simply updates the batch status and saves to the database
func (j *SetBatchStatus) Process(*config.Config) bool {
	j.DBBatch.Status = j.db.Args[bsArg]
	var err = j.DBBatch.Save()
	if err != nil {
		j.Logger.Errorf("Unable to update status for batch %d: %s", j.DBBatch.ID, err)
	}
	return err == nil
}

// SetBatchLocation is a simple job to update a batch location after files are
// copied or movied somewhere
type SetBatchLocation struct {
	*BatchJob
}

// Process just updates the batch's location field
func (j *SetBatchLocation) Process(*config.Config) bool {
	j.DBBatch.Location = j.db.Args[locArg]
	var err = j.DBBatch.Save()
	if err != nil {
		j.Logger.Errorf("Error setting batch.location for id %d: %s", j.DBBatch.ID, err)
		return false
	}

	return true
}

// RecordIssueAction adds an issue action to the Issue in question.  This one
// is slightly more involved than most metadata jobs, but in the end it's just
// a quick SQL INSERT, and an action, in my mind, really is just barely outside
// the traditional definition of metadata....
type RecordIssueAction struct {
	*IssueJob
}

// Process adds the issue action to the database
func (j *RecordIssueAction) Process(*config.Config) bool {
	// This is a waste of cycles right here, but going through the Issue's save
	// procedure ensures that the action is created and associated with the issue
	// in a way that is consistent.  If we add things to how issues and actions
	// interact, we don't really want to duplicate (or else potentially break)
	// this consistency.  Oh... and I'm lazy.
	var err = j.DBIssue.Save(models.ActionTypeInternalProcess, models.SystemUser.ID, j.db.Args[msgArg])
	if err != nil {
		j.Logger.Errorf("Error recording internal issue action for id %d: %s", j.DBIssue.ID, err)
		return false
	}

	return true
}
