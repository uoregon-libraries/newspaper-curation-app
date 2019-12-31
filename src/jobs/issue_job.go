package jobs

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// IssueJob wraps the Job type to add things needed in all jobs tied to
// specific issues
type IssueJob struct {
	*Job
	Issue            *schema.Issue
	DBIssue          *db.Issue
	updateWorkflowCB func()
}

// NewIssueJob setups up an IssueJob from a database Job, centralizing the
// common validations and data manipulation
func NewIssueJob(dbJob *db.Job) *IssueJob {
	var dbi, err = db.FindIssue(dbJob.ObjectID)
	if err != nil {
		logger.Criticalf("Unable to find issue for job %d: %s", dbJob.ID, err)
		return nil
	}

	var si *schema.Issue
	si, err = dbi.SchemaIssue()
	if err != nil {
		logger.Criticalf("Unable to prepare a schema.Issue for database issue %d: %s", dbi.ID, err)
		return nil
	}

	return &IssueJob{
		Job:     NewJob(dbJob),
		DBIssue: dbi,
		Issue:   si,
	}
}

// UpdateWorkflow calls updateWorkflowCB if defined, and then the issue job is
// saved.  At this point, however, the job is complete, so all we can do is
// loudly log failures.
func (ij *IssueJob) UpdateWorkflow() {
	if ij.updateWorkflowCB != nil {
		ij.updateWorkflowCB()
	}

	var err = ij.DBIssue.Save()
	if err != nil {
		ij.Logger.Criticalf("Unable to update issue (dbid %d) workflow post-job: %s", ij.DBIssue.ID, err)
	}
}
