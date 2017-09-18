package jobs

import (
	"config"
	"fileutil"
	"logger"
	"os"
	"path/filepath"
)

// SFTPIssueMover is a job that gets queued up when an SFTP issue is considered
// ready for processing.  It moves the issue to the workflow area and, upon
// success, queues up a page split job.
type SFTPIssueMover struct {
	*IssueJob
}

// Process moves the SFTP issue directory to the workflow area
func (im *SFTPIssueMover) Process(config *config.Config) bool {
	var iKey = im.Issue.Key()

	// Verify new path will work
	var oldLocation = im.Location
	var newLocation = filepath.Join(config.WorkflowPath, iKey)
	if !fileutil.MustNotExist(newLocation) {
		im.Logger.Error("Destination %q already exists for issue %q", newLocation, iKey)
		return false
	}

	// Move the issue directory to the workflow path
	var wipLocation = newLocation + "-wip"
	os.MkdirAll(filepath.Dir(wipLocation), 0700)
	im.Logger.Info("Copying %q to %q", oldLocation, wipLocation)
	var err = fileutil.CopyDirectory(oldLocation, wipLocation)
	if err != nil {
		im.Logger.Error("Unable to copy issue %q directory: %s", iKey, err)
		return false
	}
	err = os.RemoveAll(oldLocation)
	if err != nil {
		im.Logger.Error("Unable to clean up issue %q after copying to WIP directory: %s", iKey, err)
		return false
	}
	err = os.Rename(wipLocation, newLocation)
	if err != nil {
		im.Logger.Error("Unable to rename WIP issue directory (%q -> %q) post-copy: %s", wipLocation, newLocation, err)
		return false
	}
	im.Issue.Location = newLocation

	// Queue a new page-split job.  The SFTPIssueMover process is considered a
	// success at this point, as the move is complete, so failure to queue the
	// new job just has to be logged loudly.
	err = QueuePageSplit(im.DBIssue, im.Issue.Location)
	if err != nil {
		// NOTE: This is *not* attached to the sftp mover because the ability to
		// queue a new job isn't relevant to the completed job
		// TODO: Maybe critical logging should also be emailed somewhere
		logger.Critical("Unable to queue new page-split job!")
	}

	return true
}
