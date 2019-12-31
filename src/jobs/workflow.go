package jobs

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
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
	var err = j.DBIssue.Save()
	if err != nil {
		j.Logger.Errorf("Unable to update workflow step for issue %d: %s", j.DBIssue.ID, err)
	}
	return err == nil
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
