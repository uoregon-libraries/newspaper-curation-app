package jobs

import (
	"fmt"

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
	var j, err = newIssueJob(dbJob)
	if err != nil {
		logger.Criticalf("Unable to create issue job %d: %s", dbJob.ID, err)
	}

	return j
}

// newIssueJob actually creates the job and returns it and possibly an error.
// This is poor architecture; the NewIssueJob function shouldn't require
// logging things that have to be looked at later in order to handle broken
// database tables or other oddities, but refactoring the whole job package is
// not something I have time for (and it does work still, so... meh?)
func newIssueJob(dbJob *models.Job) (j *IssueJob, err error) {
	j = &IssueJob{Job: NewJob(dbJob)}
	j.DBIssue, err = models.FindIssue(dbJob.ObjectID)
	if err != nil {
		return j, err
	}
	if j.DBIssue == nil {
		return j, fmt.Errorf("issue id %d does not exist", dbJob.ObjectID)
	}

	j.Issue, err = j.DBIssue.SchemaIssue()
	return j, err
}

// Valid returns true if the job has a database issue and a schema issue
func (j *IssueJob) Valid() bool {
	return j.DBIssue != nil && j.Issue != nil
}
