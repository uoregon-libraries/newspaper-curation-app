// fs_jobs.go contains the simple filesystem-based jobs we want to define as
// generic jobs that many processes can use

package jobs

import (
	"os"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
)

// SyncDir is a job strictly for copying everything from one directory to
// another.  This is typically meant to be used as the first step in a "move"
// operation.  It's idempotent as well as being efficient, as it syncs files
// much like a mini-rsync, rather than doing a full copy of everything
// regardless of existing files.
type SyncDir struct {
	*Job
}

// Process does a sync from j.Source to j.Dest, only writing files that don't
// exist in j.Dest or which are different
func (j *SyncDir) Process(c *config.Config) bool {
	j.Logger.Warnf("SyncDir.Process is not implemented")
	return false
}

// UpdateWorkflow is a no-op for syncing dirs
func (j *SyncDir) UpdateWorkflow() {
}

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

// CleanFiles attempts to remove any cruft left behind from Bridge, Mac Finder,
// or other sources that hate me
type CleanFiles struct {
	*Job
}

// Process runs the file cleaner against the job's location
func (j *CleanFiles) Process(*config.Config) bool {
	var loc = j.db.Args[locArg]

	var dotfiles, err = fileutil.FindIf(loc, func(i os.FileInfo) bool {
		return !i.IsDir() && i.Name() != "" && i.Name()[0] == '.'
	})
	if err != nil {
		j.Logger.Errorf("Unable to scan for files to delete: %s", err)
		return false
	}

	for _, f := range dotfiles {
		err = os.Remove(f)
		if err != nil {
			j.Logger.Errorf("Unable to remove file %q: %s", f, err)
			return false
		}
	}

	return true
}

// UpdateWorkflow is a no-op for file cleaning
func (j *CleanFiles) UpdateWorkflow() {
}
