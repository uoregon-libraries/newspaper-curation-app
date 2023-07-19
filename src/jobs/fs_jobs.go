// fs_jobs.go contains the simple filesystem-based jobs we want to define as
// generic jobs that many processes can use

package jobs

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
)

// VerifyRecursive is a job that technically copies and verifies all files
// recursively from a source to a destination directory, but it's meant to be
// used as the final "move" step, after a faster copy operation is done. This
// process should catch any files which weren't copied properly (network
// filesystems can go to hell), and it's meant to run long enough after the
// copy that disk caching won't be likely to report false positives.
type VerifyRecursive struct {
	*Job
}

// Valid is always true since filesystem jobs identify their errors nicely, and
// FS problems that would cause real problems in Process will also be problems
// here (e.g., trying to validate the existence of a directory on an NFS mount
// that dropped)
func (j *VerifyRecursive) Valid() bool {
	return true
}

// Process does a sync from j.Source to j.Dest, only writing files that don't
// exist in j.Dest or which are different (different determined by our fileutil
// package, which is using SHA256 to test file integrity). Excluded files are
// of course neither checked nor copied.
func (j *VerifyRecursive) Process(*config.Config) bool {
	var src = j.db.Args[JobArgSource]
	var dst = j.db.Args[JobArgDestination]
	var exclusions = strings.Split(j.db.Args[JobArgExclude], ",")

	var parent = filepath.Dir(dst)
	j.Logger.Infof("Creating parent dir %q", parent)
	var err = os.MkdirAll(parent, 0700)
	if err != nil {
		j.Logger.Errorf("Unable to create sync dir's parent %q: %s", parent, err)
		return false
	}

	// We re-join exclusions here so logs show what this job will actually do,
	// which *should* be the same as what was requested, but could be different
	// if something is busted
	j.Logger.Infof("Syncing %q to %q. Exclusion list: %q", src, dst, strings.Join(exclusions, ","))
	err = fileutil.SyncDirectoryExcluding(src, dst, exclusions)
	if err != nil {
		j.Logger.Errorf("Unable to sync %q to %q: %s", src, dst, err)
	}

	return err == nil
}

// KillDir is a job to clean up an old directory, typically after a sync job
// has succeeded.
type KillDir struct {
	*Job
}

// Valid is always true since filesystem jobs identify their errors nicely, and
// FS problems that would cause real problems in Process will also be problems
// here (e.g., trying to validate the existence of a directory on an NFS mount
// that dropped)
func (j *KillDir) Valid() bool {
	return true
}

// Process removes files from j.Dir
func (j *KillDir) Process(*config.Config) bool {
	var loc = j.db.Args[JobArgLocation]
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

// RenameDir renames a directory - for the .wip-* dirs we still have to manage
// since a handful of dirs still have to be exposed to end users
type RenameDir struct {
	*Job
}

// Valid is always true since filesystem jobs identify their errors nicely, and
// FS problems that would cause real problems in Process will also be problems
// here (e.g., trying to validate the existence of a directory on an NFS mount
// that dropped)
func (j *RenameDir) Valid() bool {
	return true
}

// Process moves the source dir to the destination name
func (j *RenameDir) Process(*config.Config) bool {
	var src = j.db.Args[JobArgSource]
	var dest = j.db.Args[JobArgDestination]
	var err = os.Rename(src, dest)
	if err != nil {
		j.Logger.Errorf("Unable to rename directory (%q -> %q): %s", src, dest, err)
		return false
	}

	return true
}

// CleanFiles attempts to remove any cruft left behind from Bridge, Mac Finder,
// or other sources that hate me
type CleanFiles struct {
	*Job
}

// Valid is always true since filesystem jobs identify their errors nicely, and
// FS problems that would cause real problems in Process will also be problems
// here (e.g., trying to validate the existence of a directory on an NFS mount
// that dropped)
func (j *CleanFiles) Valid() bool {
	return true
}

// isFraggable returns true if the given file can be removed without anybody
// noticing.  This includes dotFiles, Thumbs.db, and maybe a few other random
// tidbits that get auto-created and which don't belong in an issue/batch.
func isFraggable(i os.FileInfo) bool {
	// Dirs never get fried even if they're totally useless - we can't count on
	// them being safe to delete
	if i.IsDir() {
		return false
	}

	// Can this actually happen?  Filesystems are too weird for me to discount
	// the possibility...
	if i.Name() == "" {
		return false
	}

	// Dotfiles are always removed - this isn't super-safe, but there are too
	// many cases where we get dotfiles we absolutely do not want.  Bridge
	// files, Mac files, heck I've even seen vim swapfiles once or twice.
	if i.Name()[0] == '.' {
		return true
	}

	// Thumbs.db has to be its own case.  Thanks, Windows.
	if strings.ToLower(i.Name()) == "thumbs.db" {
		return true
	}

	return false
}

// Process runs the file cleaner against the job's location
func (j *CleanFiles) Process(*config.Config) bool {
	var loc = j.db.Args[JobArgLocation]

	var fraggables, err = fileutil.FindIf(loc, isFraggable)
	if err != nil {
		j.Logger.Errorf("Unable to scan for files to delete: %s", err)
		return false
	}

	for _, f := range fraggables {
		err = os.Remove(f)
		if err != nil {
			j.Logger.Errorf("Unable to remove file %q: %s", f, err)
			return false
		}
	}

	return true
}

// RemoveFile is a simple job with one purpose: delete a file
type RemoveFile struct {
	*Job
}

// Valid is always true as long as there is *any* location arg. We don't check
// that it's a real file because Valid is a check to prevent panics, not to
// ensure the operation will succeed.
func (j *RemoveFile) Valid() bool {
	if j.db.Args[JobArgLocation] == "" {
		j.Logger.Errorf("RemoveFile job created with no location arg")
		return false
	}
	return true
}

// Process removes the file. If the file doesn't exist, this is considered a
// success because the location identified was likely already deleted.
func (j *RemoveFile) Process(*config.Config) bool {
	var fname = j.db.Args[JobArgLocation]

	j.Logger.Debugf("RemoveFile: attempting to remove %q", fname)

	var err = os.Remove(fname)
	if err == nil {
		j.Logger.Debugf("RemoveFile: successfully deleted %q", fname)
		return true
	}
	if os.IsNotExist(err) {
		j.Logger.Debugf("RemoveFile: %q was not present (success is implied)", fname)
		return true
	}

	j.Logger.Errorf("Unable to remove %q: %s", fname, err)
	return false
}
