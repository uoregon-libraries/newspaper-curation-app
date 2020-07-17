package jobs

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// BatchJob wraps the Job type to add things needed in all jobs tied to
// specific batches
type BatchJob struct {
	*Job
	DBBatch *models.Batch
}

// NewBatchJob setups up a BatchJob from a database Job, centralizing the
// common validations and data manipulation
func NewBatchJob(dbJob *models.Job) *BatchJob {
	var j, err = newBatchJob(dbJob)
	if err != nil {
		logger.Criticalf("Unable to create batch job %d: %s", dbJob.ID, err)
	}
	return j
}

// newBatchJob actually creates the job and returns it and possibly an error.
// This is poor architecture, etc. etc.  See newIssueJob for justification.
func newBatchJob(dbJob *models.Job) (j *BatchJob, err error) {
	j = &BatchJob{Job: NewJob(dbJob)}

	j.DBBatch, err = models.FindBatch(dbJob.ObjectID)
	if err != nil {
		return j, err
	}
	if j.DBBatch == nil {
		return j, fmt.Errorf("batch id %d does not exist", dbJob.ObjectID)
	}

	return j, nil
}

// Valid returns whether the database batch was found successfully
func (j *BatchJob) Valid() bool {
	return j.DBBatch != nil
}
