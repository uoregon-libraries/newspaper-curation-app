package jobs

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// JobJob wraps the Job type to add things needed in jobs which target *other*
// jobs. Confusing.
type JobJob struct {
	*Job
	TargetJob *models.Job
}

// NewJobJob setups up a JobJob from a database Job, centralizing the common
// validations and data manipulation
func NewJobJob(dbJob *models.Job) *JobJob {
	var j, err = newJobJob(dbJob)
	if err != nil {
		logger.Criticalf("Unable to create job-targeting job %d: %s", dbJob.ID, err)
	}

	return j
}

// newJobJob actually creates the job and returns it and possibly an error
func newJobJob(dbJob *models.Job) (j *JobJob, err error) {
	j = &JobJob{Job: NewJob(dbJob)}
	j.TargetJob, err = models.FindJob(dbJob.ObjectID)
	if err != nil {
		return j, err
	}
	if j.TargetJob == nil {
		return j, fmt.Errorf("job id %d does not exist", dbJob.ObjectID)
	}
	return j, err
}

// Valid returns true if the job has a target job
func (j *JobJob) Valid() bool {
	return j.TargetJob != nil
}
