package jobs

import (
	"fmt"
	"strconv"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/openoni"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

const (
	serverTypeStaging = "staging"
	serverTypeProd    = "production"
)

// getONIAgent attempts to set up a connection to the staging or prod ONI Agent
// based on the job's "location" arg
func getONIAgent(j *Job, c *config.Config) (*openoni.RPC, error) {
	var lookup = map[string]string{
		serverTypeProd:    c.ProductionAgentConnection,
		serverTypeStaging: c.StagingAgentConnection,
	}

	var st = j.db.Args[JobArgLocation]
	var connection = lookup[st]
	if connection == "" {
		return nil, fmt.Errorf("getONIAgent: invalid server type, or misconfiguration: location %q, connection %q", st, connection)
	}
	return openoni.New(connection)
}

type batchJobFunc func(batchname string) (jobid int64, err error)

func (j *BatchJob) queueAgentJob(name string, fn batchJobFunc) ProcessResponse {
	var jobid, err = fn(j.DBBatch.FullName())
	if err != nil {
		j.Logger.Errorf("Error calling ONI Agent: %s", err)
		return PRFailure
	}

	j.Logger.Infof("Queued %s job in %s ONI Agent: job id %d", name, j.db.Args[JobArgLocation], jobid)

	// Store the job id on the batch since loading is slow and logs sometimes
	// need to be scrutinized
	j.DBBatch.ONIAgentJobID = jobid

	// It's pretty critical that we save the batch job id since we've already
	// queued a job externally
	err = j.runCritical(func() error {
		var msg = fmt.Sprintf("sent ONI Agent the %s command", name)
		var err = j.DBBatch.Save(models.ActionTypeInternalProcess, models.SystemUser.ID, msg)
		if err != nil {
			j.Logger.Warnf("Unable to update batch data; retrying: %s", err)
			return err
		}
		return nil
	})

	if err != nil {
		j.Logger.Criticalf("Unable to update batch: %s", err)
		return PRFatal
	}

	j.Logger.Infof("Job queued, batch updated successfully")
	return PRSuccess
}

// ONILoadBatch calls an RPC to load a batch into ONI
type ONILoadBatch struct {
	*BatchJob
}

// Process sends the RPC request to an ONI Agent, requesting a batch load
func (j *ONILoadBatch) Process(c *config.Config) ProcessResponse {
	var agent, err = getONIAgent(j.BatchJob.Job, c)
	if err != nil {
		j.Logger.Errorf("Error constructing ONI RPC: %s", err)
		return PRFailure
	}

	return j.queueAgentJob("load batch", agent.LoadBatch)
}

// ONIPurgeBatch handles API calls to request a batch purge from ONI
type ONIPurgeBatch struct {
	*BatchJob
}

// Process sends the RPC request to an ONI Agent, requesting a batch purge
func (j *ONIPurgeBatch) Process(c *config.Config) ProcessResponse {
	var agent, err = getONIAgent(j.BatchJob.Job, c)
	if err != nil {
		j.Logger.Errorf("Error constructing ONI RPC: %s", err)
		return PRFailure
	}

	return j.queueAgentJob("purge batch", agent.PurgeBatch)
}

// ONIWaitForJob is a generic job to poll ONI until it reports that a given job
// on its end has completed
type ONIWaitForJob struct {
	*Job
}

// Valid is true as long as a valid (numeric, greater than zero) job id has
// been set in the ID arg
func (j *ONIWaitForJob) Valid() bool {
	var idstr = j.db.Args[JobArgID]
	var _, err = strconv.ParseUint(idstr, 10, 64)
	if err != nil {
		j.Logger.Errorf("ONIWaitForJob created with an invalid id (%q): %s", idstr, err)
		return false
	}
	return true
}

// Process connects to an ONI Agent and checks the status of the job. If it's
// complete, this job is done. If it's pending, this job quietly retries later.
func (j *ONIWaitForJob) Process(c *config.Config) ProcessResponse {
	var agent, err = getONIAgent(j.Job, c)
	if err != nil {
		j.Logger.Errorf("Error constructing ONI RPC: %s", err)
		return PRFailure
	}

	// We know the id must already be valid due to our Valid() implementation
	var jobID, _ = strconv.ParseUint(j.db.Args[JobArgID], 10, 64)

	var js openoni.JobStatus
	js, err = agent.GetJobStatus(int64(jobID))
	if err != nil {
		j.Logger.Errorf("Error calling ONI Agent: %s", err)
		return PRFailure
	}

	switch js {
	case openoni.JobStatusPending:
		j.Logger.Infof("ONI Agent reports job not started; will check later")
		return PRTryLater
	case openoni.JobStatusStarted:
		j.Logger.Infof("ONI Agent reports job not complete; will check later")
		return PRTryLater
	case openoni.JobStatusFailStart:
		j.Logger.Errorf("ONI Agent job %d failed to start", jobID)
		return PRFatal
	case openoni.JobStatusFailed:
		j.Logger.Errorf("ONI Agent job %d failed to complete", jobID)
		return PRFatal
	case openoni.JobStatusSuccessful:
		j.Logger.Infof("ONI Agent reports job completed successfully")
		return PRSuccess
	}

	j.Logger.Criticalf("Unknown value returned when requesting job status for job %d: %q", jobID, js)
	return PRFatal
}
