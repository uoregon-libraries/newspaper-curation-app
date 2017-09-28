package jobs

import (
	"config"
	"logger"
)

// MoveIssueForDerivatives is a job that gets queued up when an SFTP issue has
// had pages renamed manually, and is considered ready for derivatives.  It
// moves the issue to the workflow area and, upon success, queues up a
// derivative job.
type MoveIssueForDerivatives struct {
	*IssueJob
}

// Process moves the issue directory to the workflow area, deletes all hidden
// files (such as Adobe Bridge stuff), and queues up a derivative job
func (mifd *MoveIssueForDerivatives) Process(config *config.Config) bool {
	moveIssue(mifd.IssueJob, config.WorkflowPath)

	// Queue a new derivative job; failure here must be logged loudly, but
	// doesn't change the fact that the move process already happened
	var err = QueueMakeDerivatives(mifd.DBIssue, mifd.Issue.Location)
	if err != nil {
		logger.Critical("Unable to queue new derivative job for issue id %d: %s", mifd.DBIssue.ID, err)
	}

	return true
}
