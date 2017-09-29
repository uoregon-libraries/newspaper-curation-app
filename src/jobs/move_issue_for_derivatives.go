package jobs

import (
	"config"
	"fileutil"
	"logger"
	"os"
)

// MoveIssueForDerivatives is a job that gets queued up when an issue is ready
// for derivatives: an SFTP issue has had pages renamed manually, or a scanned
// issue has been assembled.  It moves the issue to the workflow area and, upon
// success, queues up a derivative job.
type MoveIssueForDerivatives struct {
	*IssueJob
}

// Process moves the issue directory to the workflow area, deletes all hidden
// files (such as Adobe Bridge stuff), and queues up a derivative job
func (mifd *MoveIssueForDerivatives) Process(config *config.Config) bool {
	if !moveIssue(mifd.IssueJob, config.WorkflowPath) {
		return false
	}

	mifd.removeDotfiles()

	// Queue a new derivative job; failure here must be logged loudly, but
	// doesn't change the fact that the move process already happened
	var err = QueueMakeDerivatives(mifd.DBIssue, mifd.Issue.Location)
	if err != nil {
		logger.Critical("Unable to queue new derivative job for issue id %d: %s", mifd.DBIssue.ID, err)
	}

	return true
}

// removeDotfiles attempts to remove any cruft left behind from Bridge, Mac
// Finder, or other sources that hate me
func (mifd *MoveIssueForDerivatives) removeDotfiles() {
	var dotfiles, err = fileutil.FindIf(mifd.Issue.Location, func(i os.FileInfo) bool {
		return !i.IsDir() && i.Name() != "" && i.Name()[0] == '.'
	})
	if err != nil {
		logger.Error("Unable to scan for files to delete: %s", err)
		return
	}

	for _, f := range dotfiles {
		err = os.Remove(f)
		if err != nil {
			logger.Error("Unable to remove file %q: %s", f, err)
		}
	}
}
