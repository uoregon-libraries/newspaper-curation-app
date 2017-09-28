package jobs

import (
	"config"
	"logger"
)

// SFTPIssueMover is a job that gets queued up when an SFTP issue is considered
// ready for processing.  It moves the issue to the workflow area and, upon
// success, queues up a page split job.
type SFTPIssueMover struct {
	*IssueJob
}

// Process moves the SFTP issue directory to the workflow area
func (im *SFTPIssueMover) Process(config *config.Config) bool {
	if !moveIssue(im.IssueJob, config.WorkflowPath) {
		return false
	}

	// Queue a new page-split job.  The SFTPIssueMover process is considered a
	// success at this point, as the move is complete, so failure to queue the
	// new job just has to be logged loudly.
	var err = QueuePageSplit(im.DBIssue, im.Issue.Location)
	if err != nil {
		// NOTE: This is *not* attached to the sftp mover because the ability to
		// queue a new job isn't relevant to the completed job
		// TODO: Maybe critical logging should also be emailed somewhere
		logger.Critical("Unable to queue new page-split job for issue id %d: %s", im.DBIssue.ID, err)
	}

	return true
}
