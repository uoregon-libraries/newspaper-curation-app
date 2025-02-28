package jobs

import (
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// FinalizeBatchFlaggedIssue is responsible for database changes needed to mark
// the issue as (a) no longer in a batch, and (b) needing user action (set to
// being an unfixable-error issue)
type FinalizeBatchFlaggedIssue struct {
	*IssueJob
}

// Process gets the issue ready for the unfixable-error state:
//   - Clear batch id and workflow owner / expiry
//   - Create two action log entries: one for removing the issue from the batch,
//     one for the error message, attributed to the user who flagged the issue
//
// This function has copious error checking and logging. While technically
// unnecessary (the DB operation uses transactions and defers errors), it
// should help debugging if necessary.
func (j *FinalizeBatchFlaggedIssue) Process(*config.Config) ProcessResponse {
	var i = j.DBIssue
	j.Logger.Debugf("Finalizing issue (%d / %s) flagged when QCing its batch (batch id %d)", i.ID, i.Key(), i.BatchID)

	// This requires transactions: we have to delete the flagged issue data from
	// the database, save the issue, and record two action logs
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()

	// Log batch-remove action
	var a = models.NewIssueAction(i.ID, models.ActionTypeInternalProcess)
	a.UserID = models.EmptyUser.ID
	a.Message = "removed from batch due to errors"
	var err = a.SaveOp(op)
	if err != nil {
		j.Logger.Errorf("Unable to create 'removed from batch' issue action: %s", err)
		op.Rollback()
		return PRFailure
	}
	j.Logger.Debugf("Successfully created 'removed from batch' issue action")

	// Update database: clear batch, workflow metadata, and set status
	var oldBatchID = i.BatchID
	i.BatchID = 0
	i.WorkflowOwnerID = 0
	i.WorkflowOwnerExpiresAt = time.Time{}
	i.WorkflowStep = schema.WSUnfixableMetadataError
	err = i.SaveOpWithoutAction(op)
	if err != nil {
		j.Logger.Errorf("Unable to clear batch and workflow data: %s", err)
		op.Rollback()
		return PRFailure
	}
	j.Logger.Debugf("Successfully cleared batch and workflow data")

	// Find flagged issue to get the error message
	var flagged *models.FlaggedIssue
	flagged, err = models.FindFlaggedIssue(oldBatchID, i.ID)
	if err != nil {
		j.Logger.Errorf("Unable to find flagged issue by id %d: %s", i.ID, err)
		op.Rollback()
		return PRFailure
	}

	// Log user's flagged-for-removal error
	a = models.NewIssueAction(i.ID, models.ActionTypeReportUnfixableError)
	a.UserID = flagged.User.ID
	a.Message = flagged.Reason
	err = a.SaveOp(op)
	if err != nil {
		j.Logger.Errorf("Unable to create removal reason issue action: %s", err)
		op.Rollback()
		return PRFailure
	}
	j.Logger.Debugf("Successfully added removal reason to issue actions")

	// Commit the transaction and check one last time for errors
	op.EndTransaction()
	if op.Err() != nil {
		j.Logger.Errorf("Database error ending transaction: %s", op.Err())
		op.Rollback()
		return PRFailure
	}

	j.Logger.Debugf("Successfully finalized issue")
	return PRSuccess
}

// EmptyBatchFlaggedIssuesList is a simple job to clear the
// batches_flagged_issues table of entries related to this job's batch
type EmptyBatchFlaggedIssuesList struct {
	*BatchJob
}

// Process just executes AbortIssueFlagging to clear the table
func (j *EmptyBatchFlaggedIssuesList) Process(*config.Config) ProcessResponse {
	j.Logger.Debugf("Removing issues flagged for removal from batch %d (%s)", j.DBBatch.ID, j.DBBatch.Name)
	var err = j.DBBatch.EmptyFlaggedIssuesList()
	if err != nil {
		j.Logger.Errorf("Database error clearing table: %s", err)
		return PRFailure
	}
	return PRSuccess
}

// DeleteBatch removes all issues from a batch and flags it as deleted
type DeleteBatch struct {
	*BatchJob
}

// Process just runs batch.Delete. Easy!
func (j *DeleteBatch) Process(*config.Config) ProcessResponse {
	j.Logger.Debugf("Removing issues and deleting batch %d (%s)", j.DBBatch.ID, j.DBBatch.Name)
	var err = j.DBBatch.Delete()
	if err != nil {
		j.Logger.Errorf("Database error deleting batch: %s", err)
		return PRFailure
	}
	return PRSuccess
}
