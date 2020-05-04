package jobs

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// IssueJob wraps the Job type to add things needed in all jobs tied to
// specific issues
type IssueJob struct {
	*Job
	Issue   *schema.Issue
	DBIssue *models.Issue
}

// NewIssueJob setups up an IssueJob from a database Job, centralizing the
// common validations and data manipulation
func NewIssueJob(dbJob *models.Job) *IssueJob {
	var dbi, err = models.FindIssue(dbJob.ObjectID)
	if err != nil {
		logger.Criticalf("Unable to find issue for job %d: %s", dbJob.ID, err)
		return nil
	}
	if dbi == nil {
		logger.Criticalf("No issue exists for job %d (issue id %d)", dbJob.ID, dbJob.ObjectID)
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
