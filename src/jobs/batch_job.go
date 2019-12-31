package jobs

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

// BatchJob wraps the Job type to add things needed in all jobs tied to
// specific batches
type BatchJob struct {
	*Job
	DBBatch          *db.Batch
	updateWorkflowCB func()
}

// NewBatchJob setups up a BatchJob from a database Job, centralizing the
// common validations and data manipulation
func NewBatchJob(dbJob *db.Job) *BatchJob {
	var batch, err = db.FindBatch(dbJob.ObjectID)
	if err != nil {
		logger.Criticalf("Unable to find batch for job %d: %s", dbJob.ID, err)
		return nil
	}

	return &BatchJob{Job: NewJob(dbJob), DBBatch: batch, updateWorkflowCB: nilWorkflowCB}
}

// nilWorkflowCB just lets us have a placeholder for the workflow callback in
// cases the implementor doesn't set one
func nilWorkflowCB() {
}

// UpdateWorkflow implements Processor, calling the updateWorkflowCB if any,
// then saving the batch back to the database
func (j *BatchJob) UpdateWorkflow() {
	j.updateWorkflowCB()
	var err = j.DBBatch.Save()
	if err != nil {
		j.Logger.Criticalf("Unable to update batch (dbid %d) post-job: %s", j.DBBatch.ID, err)
	}
}
