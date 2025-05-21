package jobs

import (
	"fmt"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/openoni"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/retry"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
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
	var server = j.db.Args[JobArgLocation]
	if server == "" {
		return nil, fmt.Errorf("getONIAgent: server type (location arg) is required")
	}

	var st = j.db.Args[JobArgLocation]
	var connection = lookup[st]
	if connection == "" {
		return nil, fmt.Errorf("getONIAgent: config error: server type (location arg) %q has empty connection string", st)
	}
	return openoni.New(connection)
}

type batchJobFunc func(batchname string) (jobid int64, err error)

func (j *BatchJob) queueAgentJob(name string, fn batchJobFunc) ProcessResponse {
	var jobid, err = fn(j.DBBatch.FullName)
	if err != nil {
		j.Logger.Errorf("Error calling ONI Agent: %s", err)
		return PRFailure
	}

	j.Logger.Infof("Queued %s job in %s ONI Agent: job id %d", name, j.db.Args[JobArgLocation], jobid)

	// Store the job id on the batch since loading is slow and logs sometimes
	// need to be scrutinized
	j.DBBatch.ONIAgentJobID = jobid

	// It's pretty critical that we save the batch data since the ONI job was
	// successfully created
	err = retry.Do(time.Minute*10, func() error {
		var msg = fmt.Sprintf("Sent ONI Agent the %s command", name)
		var err = j.DBBatch.Save(models.ActionTypeInternalProcess, models.SystemUser.ID, msg)
		if err != nil {
			j.Logger.Warnf("Unable to update batch data. Retrying: %s", err)
			return err
		}

		j.Logger.Infof(msg)
		return nil
	})

	if err != nil {
		j.Logger.Criticalf("Unable to update batch after successfully queueing an ONI job! Manual intervention required! Error: %s", err)
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

	var moc *models.MOC
	var code = j.BatchJob.DBBatch.MARCOrgCode
	moc, err = models.FindMOCByCode(code)
	if err != nil {
		j.Logger.Errorf("Error looking up MOC %q: %s", code, err)
		return PRFailure
	}
	if moc == nil {
		j.Logger.Errorf("Error looking up MOC %q: no such code exists", code)
		return PRFatal
	}

	var msg string
	msg, err = agent.EnsureAwardee(moc)
	if err != nil {
		j.Logger.Errorf("ONI Agent couldn't verify awardee's existence: %s", err)
		return PRFailure
	}
	j.Logger.Infof("ONI Agent ensure-awardee response: %s", msg)
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

// ONIWaitForJob is a generic job to poll ONI until it reports that a given
// batch's job on its end has completed. This is currently specific to batches.
// If we need it to work with other things, we'll have to add something to the
// pipeline allowing a job to tell the future job which ONI id to wait for.
type ONIWaitForJob struct {
	*BatchJob
}

// Valid is true as long as a valid (numeric, greater than zero) job id has
// been set in the ID arg
func (j *ONIWaitForJob) Valid() bool {
	var id = j.DBBatch.ONIAgentJobID
	if id == -1 {
		j.Logger.Infof("ONIWaitForJob done: Agent reported unnecessary job")
		return true
	}
	if id < 1 {
		j.Logger.Errorf("ONIWaitForJob created with an invalid id (%d)", id)
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

	var jobID = j.DBBatch.ONIAgentJobID

	var js openoni.JobStatus
	js, err = agent.GetJobStatus(int64(jobID))
	if err != nil {
		j.Logger.Errorf("Error calling ONI Agent: %s", err)
		return PRFailure
	}

	switch js {
	case openoni.JobStatusPending:
		j.Logger.Infof("ONI Agent reports job not started. Will check later")
		return PRTryLater
	case openoni.JobStatusStarted:
		j.Logger.Infof("ONI Agent reports job not complete. Will check later")
		return PRTryLater
	case openoni.JobStatusFailStart:
		j.Logger.Errorf("ONI Agent job %d failed to start", jobID)
		return PRFailure
	case openoni.JobStatusFailed:
		j.Logger.Errorf("ONI Agent job %d failed to complete", jobID)
		return PRFailure
	case openoni.JobStatusSuccessful:
		j.Logger.Infof("ONI Agent reports job completed successfully")
		return PRSuccess
	}

	j.Logger.Errorf("Unknown value returned when requesting job status for job %d: %q", jobID, js)
	return PRFatal
}
