package jobs

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/openoni"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

const (
	serverTypeStaging = "staging"
	serverTypeProd    = "production"
)

func getServerURL(j *Job, c *config.Config) string {
	var st = j.db.Args[JobArgLocation]
	switch st {
	case serverTypeProd:
		return c.NewsWebroot

	case serverTypeStaging:
		return c.StagingNewsWebroot

	default:
		j.Logger.Errorf("Invalid server type(%q: %q)", JobArgLocation, st)
		return ""
	}
}

// ONILoadBatch calls an RPC to load a batch into ONI
type ONILoadBatch struct {
	*BatchJob
}

// Process sends the RPC request and handles its response, then kicks off a new
// job to poll ONI and wait for its work to be done
func (j *ONILoadBatch) Process(c *config.Config) ProcessResponse {
	var serverURL = getServerURL(j.Job, c)
	if serverURL == "" {
		j.Logger.Errorf("Unable to determine server URL")
		return PRFailure
	}

	// We want to get the pipeline *before* sending a request ONI. If there are
	// temporary database errors, we're hoping to catch that here rather than
	// after we've started a job externally.
	var _, err = models.FindPipeline(j.DBJob().PipelineID)
	if err != nil {
		j.Logger.Errorf("Cannot read job's pipeline from the database: %s", err)
		return PRFailure
	}

	_, err = openoni.New(serverURL)
	if err != nil {
		j.Logger.Errorf("Error constructing ONI RPC: %s", err)
		return PRFailure
	}

	// TODO: handle response
	//   - Response of 409, retry later
	//   - Other 4xx response, critical failure (no retry), log data
	//   - 5xx response, report temporary failure, log data returned
	//   - 2xx response, spawn new job ("wait for API process to complete") and return success
	//   - General HTTP error, temporary failure, log
	// api.LoadBatch(j.DBBatch.FullName())

	// TODO: Queue up a new job to wait for ONI - if this returns an error, what
	// do we do?
	// - Make it a critical failure with no retry to avoid rerunning this job?
	//   Would hitting ONI a second time with the same batch cause any major
	//   failures overall?
	// - Normal failure and find a way to handle it if the batch is already
	//   queueing / ingested into ONI?
	var waitJob = models.NewJob(models.JobTypeONIWaitForJob, makeSrcArgs("foo"))
	err = j.db.QueueSiblingJobs([]*models.Job{waitJob})
	if err != nil {
		j.Logger.Errorf(`Unable to queue "ONIWaitForJob" job: %s`, err)
		return PRFailure
	}

	j.Logger.Errorf("Not implemented; skipping API call")
	return PRSuccess
}

// ONIPurgeBatch handles API calls to request a batch purge from ONI
type ONIPurgeBatch struct {
	*BatchJob
}

// Process connects to ONI and requests a batch be purged
func (j *ONIPurgeBatch) Process(c *config.Config) ProcessResponse {
	var serverURL = getServerURL(j.Job, c)
	if serverURL == "" {
		j.Logger.Errorf("Unable to determine server URL")
		return PRFailure
	}

	var _, err = openoni.New(serverURL)
	if err != nil {
		j.Logger.Errorf("Error constructing ONI RPC: %s", err)
		return PRFailure
	}

	j.Logger.Errorf("Not implemented; skipping API call")
	return PRSuccess

	// TODO: Queue up a new job to wait for ONI - make an "insert" function in
	// pipelines to add it after this job, pushing everything else out one level
}

// ONIWaitForJob is a generic job to poll ONI until it reports that a given job
// on its end has completed
type ONIWaitForJob struct {
	*Job
}

// Valid is always true for simplicity; it should be impossible to build a
// broken job, and any problems will be found in Process() anyway
func (j *ONIWaitForJob) Valid() bool {
	return true
}

// Process connects to ONI and checks the status of the job. If it's complete,
// this job is done. If it's pending, this job quietly retries later.
func (j *ONIWaitForJob) Process(c *config.Config) ProcessResponse {
	var serverURL = getServerURL(j.Job, c)
	if serverURL == "" {
		j.Logger.Errorf("Unable to determine server URL")
		return PRFailure
	}

	var _, err = openoni.New(serverURL)
	if err != nil {
		j.Logger.Errorf("Error constructing ONI RPC: %s", err)
		return PRFailure
	}

	j.Logger.Errorf("Not implemented; skipping API call")
	return PRSuccess
}
