package jobs

import (
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
	var batch, err = models.FindBatch(dbJob.ObjectID)
	if err != nil {
		logger.Criticalf("Unable to find batch for job %d: %s", dbJob.ID, err)
		return nil
	}

	return &BatchJob{Job: NewJob(dbJob), DBBatch: batch}
}
