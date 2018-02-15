package jobs

import (
	"config"
)

// PageReviewIssueMover is a job that gets queued up when an issue needs to get
// into the page review area for manual processing
type PageReviewIssueMover struct {
	*IssueJob
}

// Process moves the issue directory to the page review area
func (job *PageReviewIssueMover) Process(config *config.Config) bool {
	return moveIssue(job.IssueJob, config.PDFPageReviewPath)
}
