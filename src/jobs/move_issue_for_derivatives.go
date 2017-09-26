package jobs

import (
	"config"
	"fileutil"
	"logger"
	"os"
	"path/filepath"
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
	var iKey = mifd.Issue.Key()

	// Verify new path will work
	var oldLocation = mifd.Location
	var newLocation = filepath.Join(config.WorkflowPath, mifd.Subdir())
	if !fileutil.MustNotExist(newLocation) {
		mifd.Logger.Error("Destination %q already exists for issue %q", newLocation, iKey)
		return false
	}

	// Move the issue directory to the workflow path
	var wipLocation = filepath.Join(config.WorkflowPath, mifd.WIPDir())
	mifd.Logger.Info("Copying %q to %q", oldLocation, wipLocation)
	var err = fileutil.CopyDirectory(oldLocation, wipLocation)
	if err != nil {
		mifd.Logger.Error("Unable to copy issue %q directory: %s", iKey, err)
		return false
	}
	err = os.RemoveAll(oldLocation)
	if err != nil {
		mifd.Logger.Error("Unable to clean up issue %q after copying to WIP directory: %s", iKey, err)
		return false
	}
	err = os.Rename(wipLocation, newLocation)
	if err != nil {
		mifd.Logger.Error("Unable to rename WIP issue directory (%q -> %q) post-copy: %s", wipLocation, newLocation, err)
		return false
	}
	mifd.Issue.Location = newLocation

	// Queue a new derivative job; failure here must be logged loudly, but
	// doesn't change the fact that the move process already happened
	err = QueueMakeDerivatives(mifd.DBIssue, mifd.Issue.Location)
	if err != nil {
		logger.Critical("Unable to queue new derivative job for issue id %d: %s", mifd.DBIssue.ID, err)
	}

	// Update workflow info; failure here must be logged loudly, but doesn't
	// change the fact that the move process already happened
	mifd.DBIssue.Location = newLocation
	err = mifd.DBIssue.Save()
	if err != nil {
		mifd.Logger.Critical("Unable to update Issue location for id %d: %s", mifd.DBIssue.ID, err)
	}

	return true
}
