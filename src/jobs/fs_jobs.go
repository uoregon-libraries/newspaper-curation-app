// fs_jobs.go contains the simple filesystem-based jobs we want to define as
// generic jobs that many processes can use

package jobs

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/fileutil/manifest"
	"github.com/uoregon-libraries/gopkg/hasher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// SyncRecursive is a very special type of job that reads everything in a given
// dir, copying files as it goes, and spawning new jobs whenever a subdir is
// found. A file is synced with minimal verification, just ensuring that the
// operating system didn't return any errors. This should generally be followed
// up with a VerifyRecursive operation due to issues which can occur when a
// mounted filesystem has "hiccups".
type SyncRecursive struct {
	*Job
}

// Valid is always true since filesystem jobs identify their errors nicely, and
// FS problems that would cause real problems in Process will also be problems
// here (e.g., trying to validate the existence of a directory on an NFS mount
// that dropped)
func (j *SyncRecursive) Valid() bool {
	return true
}

func (j *SyncRecursive) isExcluded(path string, exclusions []string) bool {
	var basename = filepath.Base(path)
	for _, pattern := range exclusions {
		var match, err = filepath.Match(pattern, basename)
		// For simplicity, a bad pattern will log an error and return false, but
		// allow the processing to otherwise continue.
		if err != nil {
			j.Logger.Errorf("Error checking %q against pattern %q: %s", path, pattern, err)
			return false
		}
		if match {
			return true
		}
	}

	return false
}

// Process does a sync from j.Source to j.Dest, only writing files that don't
// exist in j.Dest or which have a different size. Excluded files are of course
// neither checked nor copied.
func (j *SyncRecursive) Process(*config.Config) ProcessResponse {
	var src = j.db.Args[JobArgSource]
	var dst = j.db.Args[JobArgDestination]
	var exclusions = strings.Split(j.db.Args[JobArgExclude], ",")
	var err error

	j.Logger.Infof("Copying %q to %q excluding %q", src, dst, strings.Join(exclusions, ","))
	var srcInfo os.FileInfo
	srcInfo, err = os.Stat(src)
	if err != nil {
		j.Logger.Errorf("Unable to stat directory %q: %s", src, err)
		return PRFailure
	}

	// Create dest dir with same permissions as source dir
	var srcMode = srcInfo.Mode() & os.ModePerm
	err = os.MkdirAll(dst, srcMode)
	if err != nil {
		j.Logger.Errorf("Unable to create destination directory %q: %s", dst, err)
		return PRFailure
	}

	// Just in case dir was already there, we force the permissions
	err = os.Chmod(dst, srcMode)
	if err != nil {
		j.Logger.Errorf("Unable to create destination directory %q: %s", dst, err)
		return PRFailure
	}

	var entries []fs.DirEntry
	entries, err = os.ReadDir(src)
	if err != nil {
		j.Logger.Errorf("Unable to read directory %q: %s", src, err)
		return PRFailure
	}

	// We build, but don't save, all dir copy jobs so we can first copy all
	// files, and only on success queue up jobs. This *should* prevent any duped
	// jobs because we can't fail after the dirs are jobbed up in a transaction.
	var dirJobs []*models.Job

	for _, entry := range entries {
		var srcFull = filepath.Join(src, entry.Name())
		var dstFull = filepath.Join(dst, entry.Name())
		var info, err = entry.Info()
		if err != nil {
			j.Logger.Errorf("Unable to read file %q: %s", entry.Name(), err)
			return PRFailure
		}

		switch {
		case info.Mode().IsRegular():
			if j.isExcluded(srcFull, exclusions) {
				j.Logger.Debugf("Found file %q, skipping per exclusion list", srcFull)
			} else {
				j.Logger.Debugf("Found file %q, copying", srcFull)
				err = syncFileFast(srcFull, dstFull)
				if err != nil {
					j.Logger.Errorf("Unable to copy %q to %q: %s", srcFull, dstFull, err)
					return PRFailure
				}
			}

		case info.Mode().IsDir():
			j.Logger.Infof("Found subdirectory %q, preparing new job", srcFull)
			var args = makeSrcDstArgs(srcFull, dstFull)
			args[JobArgExclude] = j.db.Args[JobArgExclude]
			dirJobs = append(dirJobs, models.NewJob(models.JobTypeSyncRecursive, args))

		default:
			j.Logger.Errorf("Invalid file type for %q, cannot continue copying", srcFull)
			return PRFatal
		}
	}

	err = j.db.QueueSiblingJobs(dirJobs)
	if err != nil {
		j.Logger.Errorf("Unable to queue subdir copy jobs: %s", err)
		return PRFailure
	}

	j.Logger.Infof("Fast sync successful")
	return PRSuccess
}

// syncFileFast copies src file to dst if either dst doesn't exist or is a
// different size than src. No validation is done after copying, other than
// that there were no OS errors returned.
func syncFileFast(src, dst string) error {
	if fileutil.MustNotExist(dst) {
		return fileutil.CopyFile(src, dst)
	}

	var err error
	var si, di os.FileInfo
	si, err = os.Stat(src)
	if err == nil {
		di, err = os.Stat(dst)
	}
	if err != nil {
		return err
	}
	if si.Size() != di.Size() {
		return fileutil.CopyFile(src, dst)
	}

	return nil
}

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
func (j *VerifyRecursive) Process(*config.Config) ProcessResponse {
	var src = j.db.Args[JobArgSource]
	var dst = j.db.Args[JobArgDestination]
	var exclusions = strings.Split(j.db.Args[JobArgExclude], ",")

	var parent = filepath.Dir(dst)
	j.Logger.Debugf("Creating parent dir %q", parent)
	var err = os.MkdirAll(parent, 0700)
	if err != nil {
		j.Logger.Errorf("Unable to create sync dir's parent %q: %s", parent, err)
		return PRFailure
	}

	// We re-join exclusions here so logs show what this job will actually do,
	// which *should* be the same as what was requested, but could be different
	// if something is busted
	j.Logger.Infof("Recursively verifying copy of %q to %q excluding %q", src, dst, strings.Join(exclusions, ","))
	err = fileutil.SyncDirectoryExcluding(src, dst, exclusions)
	if err != nil {
		j.Logger.Errorf("Unable to sync %q to %q: %s", src, dst, err)
		return PRFailure
	}

	j.Logger.Infof("Fast sync completed")
	return PRSuccess
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
func (j *KillDir) Process(*config.Config) ProcessResponse {
	var loc = j.db.Args[JobArgLocation]
	j.Logger.Debugf("KillDir: attempting to remove %q", loc)

	if loc == "" {
		j.Logger.Errorf("KillDir job created with no location arg")
		return PRFatal
	}
	var err = os.RemoveAll(loc)
	if err != nil {
		j.Logger.Errorf("KillDir: unable to remove %q: %s", loc, err)
		return PRFailure
	}
	return PRSuccess
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
func (j *RenameDir) Process(*config.Config) ProcessResponse {
	var src = j.db.Args[JobArgSource]
	var dest = j.db.Args[JobArgDestination]
	var err = os.Rename(src, dest)
	if err != nil {
		j.Logger.Errorf("Unable to rename directory (%q -> %q): %s", src, dest, err)
		return PRFailure
	}

	return PRSuccess
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
func (j *CleanFiles) Process(*config.Config) ProcessResponse {
	var loc = j.db.Args[JobArgLocation]

	var fraggables, err = fileutil.FindIf(loc, isFraggable)
	if err != nil {
		j.Logger.Errorf("Unable to scan for files to delete: %s", err)
		return PRFailure
	}

	for _, f := range fraggables {
		err = os.Remove(f)
		if err != nil {
			j.Logger.Errorf("Unable to remove file %q: %s", f, err)
			return PRFailure
		}
	}

	return PRSuccess
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
func (j *RemoveFile) Process(*config.Config) ProcessResponse {
	var fname = j.db.Args[JobArgLocation]

	j.Logger.Debugf("RemoveFile: attempting to remove %q", fname)

	var err = os.Remove(fname)
	if err == nil {
		j.Logger.Debugf("RemoveFile: successfully deleted %q", fname)
		return PRSuccess
	}
	if os.IsNotExist(err) {
		j.Logger.Debugf("RemoveFile: %q was not present (success is implied)", fname)
		return PRSuccess
	}

	j.Logger.Errorf("Unable to remove %q: %s", fname, err)
	return PRFailure
}

// MakeManifest is a job for creating a manifest for a directory (generally
// just for issues' files) with SHA sums
type MakeManifest struct {
	*Job
}

// Valid is true as long as we have any location arg set
func (j *MakeManifest) Valid() bool {
	if j.db.Args[JobArgLocation] == "" {
		j.Logger.Errorf("MakeManifest job created with no location arg")
		return false
	}
	return true
}

// Process creates the manifest file with a SHA256 hash
func (j *MakeManifest) Process(*config.Config) ProcessResponse {
	var dirname = j.db.Args[JobArgLocation]

	j.Logger.Debugf("MakeManifest: attempting to build manifest for %q", dirname)

	var m, err = manifest.BuildHashed(dirname, hasher.NewSHA256())
	if err != nil {
		j.Logger.Errorf("Unable to build manifest for %q", dirname)
		return PRFailure
	}

	err = m.Write()
	if err != nil {
		j.Logger.Errorf("Unable to write manifest for %q", dirname)
		return PRFailure
	}

	j.Logger.Debugf("Success")
	return PRSuccess
}
