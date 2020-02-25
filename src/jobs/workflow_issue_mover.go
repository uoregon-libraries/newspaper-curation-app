package jobs

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
)

// WorkflowIssueMover is a job that gets queued up when an issue needs to get
// into the workflow area for further processing
type WorkflowIssueMover struct {
	*IssueJob
}

// Process moves the issue directory to the workflow area
func (job *WorkflowIssueMover) Process(config *config.Config) bool {
	if !moveIssue(job.IssueJob, config.WorkflowPath) {
		return false
	}

	return true
}
