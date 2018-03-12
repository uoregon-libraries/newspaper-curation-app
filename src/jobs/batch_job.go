package jobs

import (
	"db"

	"github.com/uoregon-libraries/gopkg/logger"
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

// ObjectLocation implements the Processor interface
func (j *BatchJob) ObjectLocation() string {
	return j.DBBatch.Location
}

// nilWorkflowCB just lets us have a placeholder for the workflow callback in
// cases the implementor doesn't set one
func nilWorkflowCB() {
}

// UpdateWorkflow implements Processor, setting the batch's status if
// "ExtraData" is set.  updateWorkflowCB is then called, and the batch data
// saved back to the database.
func (j *BatchJob) UpdateWorkflow() {
	if j.ExtraData != "" {
		j.DBBatch.Status = j.ExtraData
	}

	j.updateWorkflowCB()
	var err = j.DBBatch.Save()
	if err != nil {
		j.Logger.Criticalf("Unable to update batch (dbid %d) post-job: %s", j.DBBatch.ID, err)
	}
}
