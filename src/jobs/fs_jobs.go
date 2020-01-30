// fs_jobs.go contains the simple filesystem-based jobs we want to define as
// generic jobs that many processes can use

package jobs

import (
	"os"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
)

// KillDir is a job to clean up an old directory, typically after a sync job
// has succeeded.
type KillDir struct {
	*Job
}

// Process removes files from j.Dir
func (j *KillDir) Process(c *config.Config) bool {
	var loc = j.db.Args[locArg]
	j.Logger.Debugf("KillDir: attempting to remove %q", loc)

	if loc == "" {
		j.Logger.Errorf("KillDir job created with no location arg")
		return false
	}
	var err = os.RemoveAll(loc)
	if err != nil {
		j.Logger.Errorf("KillDir: unable to remove %q: %s", loc, err)
	}
	return err == nil
}

// UpdateWorkflow is a no-op for deleting dirs
func (j *KillDir) UpdateWorkflow() {
}

// RenameDir renames a directory - for the .wip-* dirs we still have to manage
// since a handful of dirs still have to be exposed to end users
type RenameDir struct {
	*Job
}

// Process moves the source dir to the destination name
func (j *RenameDir) Process(*config.Config) bool {
	var src = j.db.Args[srcArg]
	var dest = j.db.Args[destArg]
	var err = os.Rename(src, dest)
	if err != nil {
		j.Logger.Errorf("Unable to rename directory (%q -> %q): %s", src, dest, err)
		return false
	}

	return true
}

// UpdateWorkflow is a no-op for renaming dirs
func (j *RenameDir) UpdateWorkflow() {
}
