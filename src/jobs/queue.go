package jobs

import (
	"db"
	"schema"
	"time"
)

// PrepareIssueJobAdvanced is a way to get an issue job ready with the
// necessary base values, but not save it immediately, to allow for more
// advanced job semantics: specifying that the job shouldn't run immediately,
// should queue a specific job ID after completion, should set the WorkflowStep
// to a custom value rather than whatever the job would normally do, etc.
func PrepareIssueJobAdvanced(t JobType, issue *db.Issue, path string) *db.Job {
	return &db.Job{
		Type:     string(t),
		ObjectID: issue.ID,
		Location: path,
		Status:   string(JobStatusPending),
		RunAt:    time.Now(),
	}
}

func queueIssueJob(t JobType, issue *db.Issue, path string, nextWS schema.WorkflowStep) error {
	var j = PrepareIssueJobAdvanced(t, issue, path)
	j.NextWorkflowStep = string(nextWS)
	return j.Save()
}

// QueuePageSplit creates and queues a page-splitting job with the given data
func QueuePageSplit(issue *db.Issue, path string) error {
	return queueIssueJob(JobTypePageSplit, issue, path, schema.WSAwaitingPageReview)
}

// QueueSFTPIssueMove creates an sftp issue move job
func QueueSFTPIssueMove(issue *db.Issue, path string) error {
	return queueIssueJob(JobTypeSFTPIssueMove, issue, path, schema.WSNil)
}

// QueueScanIssueMove creates a scan issue move job
func QueueScanIssueMove(issue *db.Issue, path string) error {
	return queueIssueJob(JobTypeScanIssueMove, issue, path, schema.WSNil)
}

// QueueMoveIssueForDerivatives creates and queues a job to move an issue dir
// into the workflow area so a derivative job can be created
func QueueMoveIssueForDerivatives(issue *db.Issue, path string) error {
	return queueIssueJob(JobTypeMoveIssueForDerivatives, issue, path, schema.WSNil)
}

// QueueMakeDerivatives creates and queues a job to generate ALTO XML and JP2s
// for an issue
func QueueMakeDerivatives(issue *db.Issue, path string) error {
	return queueIssueJob(JobTypeMakeDerivatives, issue, path, schema.WSReadyForMetadataEntry)
}

// QueueBuildMETS creates and queues a job to generate the METS XML for an
// issue that's been moved through the metadata queue
func QueueBuildMETS(issue *db.Issue, path string) error {
	return queueIssueJob(JobTypeBuildMETS, issue, path, schema.WSReadyForBatching)
}
