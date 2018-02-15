package jobs

import (
	"config"
	"os"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

// WorkflowIssueMover is a job that gets queued up when an issue needs to get
// into the workflow area for further processing
type WorkflowIssueMover struct {
	*IssueJob
}

// Process moves the issue directory to the workflow area, deletes all hidden
// files (such as Adobe Bridge stuff), and queues up a derivative job
func (job *WorkflowIssueMover) Process(config *config.Config) bool {
	if !moveIssue(job.IssueJob, config.WorkflowPath) {
		return false
	}

	job.removeDotfiles()

	return true
}

// removeDotfiles attempts to remove any cruft left behind from Bridge, Mac
// Finder, or other sources that hate me
func (job *WorkflowIssueMover) removeDotfiles() {
	var dotfiles, err = fileutil.FindIf(job.Issue.Location, func(i os.FileInfo) bool {
		return !i.IsDir() && i.Name() != "" && i.Name()[0] == '.'
	})
	if err != nil {
		job.Logger.Errorf("Unable to scan for files to delete: %s", err)
		return
	}

	for _, f := range dotfiles {
		err = os.Remove(f)
		if err != nil {
			job.Logger.Errorf("Unable to remove file %q: %s", f, err)
		}
	}
}
