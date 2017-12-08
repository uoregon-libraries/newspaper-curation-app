package jobs

import (
	"os"
	"path/filepath"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

// moveIssue is used by both the sftp and scan issue movers to consistently
// validate and move the source issue directory into the workflow location
func moveIssue(ij *IssueJob, path string) bool {
	var iKey = ij.Issue.Key()

	// Verify new path will work
	var oldLocation = ij.Location
	var newLocation = filepath.Join(path, ij.Subdir())
	if !fileutil.MustNotExist(newLocation) {
		ij.Logger.Errorf("Destination %q already exists for issue %q", newLocation, iKey)
		return false
	}

	// Move the issue directory to the workflow path
	var wipLocation = filepath.Join(path, ij.WIPDir())
	ij.Logger.Infof("Copying %q to %q", oldLocation, wipLocation)
	var err = fileutil.CopyDirectory(oldLocation, wipLocation)
	if err != nil {
		ij.Logger.Errorf("Unable to copy issue %q directory: %s", iKey, err)
		return false
	}
	err = os.RemoveAll(oldLocation)
	if err != nil {
		ij.Logger.Errorf("Unable to clean up issue %q after copying to WIP directory: %s", iKey, err)
		return false
	}
	err = os.Rename(wipLocation, newLocation)
	if err != nil {
		ij.Logger.Errorf("Unable to rename WIP issue directory (%q -> %q) post-copy: %s", wipLocation, newLocation, err)
		return false
	}
	ij.Issue.Location = newLocation

	// The issue has been moved, so a failure updating the record isn't a failure
	// and can only be logged loudly
	ij.DBIssue.Location = ij.Issue.Location
	err = ij.DBIssue.Save()
	if err != nil {
		ij.Logger.Criticalf("Unable to update Issue's location for id %d: %s", ij.DBIssue.ID, err)
	}

	return true
}
