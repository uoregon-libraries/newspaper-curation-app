package jobs

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/openoni"
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

	var jobid int64
	jobid, err = agent.LoadBatch(j.DBBatch.FullName())
	if err != nil {
		j.Logger.Errorf("Error calling ONI Agent: %s", err)
		return PRFailure
	}

	j.Logger.Infof("Queued load-batch job in ONI Agent: job id %d", jobid)

	// TODO: store the job id somewhere so we can monitor the job!

	return PRFailure
}
