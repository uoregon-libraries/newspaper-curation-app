package jobs

import (
	"config"
)

// SFTPIssueMover is a job that gets queued up when an SFTP issue is considered
// ready for processing.  It moves the issue to the workflow area and, upon
// success, queues up a page split job.
type SFTPIssueMover struct {
	*IssueJob
}

// Process moves the SFTP issue directory to the workflow area,
func (im *SFTPIssueMover) Process(config *config.Config) {
}
