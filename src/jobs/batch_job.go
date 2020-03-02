package jobs

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

// BatchJob wraps the Job type to add things needed in all jobs tied to
// specific batches
type BatchJob struct {
	*Job
	DBBatch *db.Batch
}

// NewBatchJob setups up a BatchJob from a database Job, centralizing the
// common validations and data manipulation
func NewBatchJob(dbJob *db.Job) *BatchJob {
	var batch, err = db.FindBatch(dbJob.ObjectID)
	if err != nil {
		logger.Criticalf("Unable to find batch for job %d: %s", dbJob.ID, err)
		return nil
	}

	return &BatchJob{Job: NewJob(dbJob), DBBatch: batch}
}
