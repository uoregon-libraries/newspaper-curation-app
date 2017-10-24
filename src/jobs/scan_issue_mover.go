package jobs

import (
	"config"
)

// ScanIssueMover is a job that gets queued up when a scanned issue is
// considered ready for processing.  It moves the issue to the workflow area
// and, upon success, queues up a derivative generation job.
type ScanIssueMover struct {
	*IssueJob
}

// Process moves the SFTP issue directory to the workflow area
func (im *ScanIssueMover) Process(config *config.Config) bool {
	if !moveIssue(im.IssueJob, config.WorkflowPath) {
		return false
	}

	// Queue a new derivative generation job.  The ScanIssueMover process is
	// considered a success at this point, as the move is complete, so failure to
	// queue the new job just has to be logged loudly.
	var err = QueueMakeDerivatives(im.DBIssue, im.Issue.Location)
	if err != nil {
		im.Logger.Criticalf("Unable to queue new derivatives job for issue id %d: %s", im.DBIssue.ID, err)
	}

	return true
}
