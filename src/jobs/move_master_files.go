package jobs

import (
	"archive/tar"
	"config"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

// MoveMasterFilesToIssueLocation is the job which finds backed-up master files
// (if any exist), creates an archive from them, copies them into the issue
// location, and then removes the masters from disk.  This only matters to
// born-digital issues; the scanned issues aren't pre-processed like PDFs are.
type MoveMasterFilesToIssueLocation struct {
	*IssueJob
}

// Process implements Processor, moving the issue's master files
func (j *MoveMasterFilesToIssueLocation) Process(c *config.Config) bool {
	if j.DBIssue.MasterBackupLocation == "" {
		j.Logger.Debugf("Master file move job for issue id %d skipped - no master files stored", j.DBIssue.ID)
		return true
	}

	j.Logger.Debugf("Starting master file move for issue id %d", j.DBIssue.ID)
	var err = j.makeMasterTar()
	if err != nil {
		j.Logger.Errorf("Unable to produce tarfile from master PDF(s): %s", err)
		return false
	}

	// Errors in the cleanup phase are tricky - we've already succeeded at
	// getting the master into the right location, so we don't want to call the
	// job a failure, but we do need to alert somebody to manually clean up the
	// master files
	err = os.RemoveAll(j.DBIssue.MasterBackupLocation)
	if err != nil {
		j.Logger.Errorf("Unable to remove master files after copy: %s.  Job is "+
			"successful, but master files need manual cleanup.", err)
	}

	// In case the issue has to be pushed back to metadata entry/review, we don't
	// want to be trying to re-archive the master files
	j.DBIssue.MasterBackupLocation = ""
	err = j.DBIssue.Save()
	if err != nil {
		j.Logger.Criticalf("Unable to update issue master backup location to '': %s", err)
	}

	j.Logger.Debugf("Master files moved successfully")
	return true
}

func (j *MoveMasterFilesToIssueLocation) makeMasterTar() error {
	var src = j.DBIssue.MasterBackupLocation
	var dst = filepath.Join(j.Location, "master")

	var f = fileutil.NewSafeFile(dst)
	defer f.Close()

	var tw = tar.NewWriter(f)
	var infos, err = ioutil.ReadDir(src)
	if err != nil {
		f.Cancel()
		return fmt.Errorf("couldn't read %q: %s", src, err)
	}

	for _, info := range infos {
		var fname = info.Name()
		j.Logger.Debugf("Writing tar header for %q", fname)
		var hdr = &tar.Header{
			Name: filepath.Join("master", fname),
			Mode: 0666,
			Size: info.Size(),
		}

		// Aggregation of errors
		var srcFile *os.File
		err = tw.WriteHeader(hdr)
		if err == nil {
			j.Logger.Debugf("Reading source file %q", fname)
			srcFile, err = os.Open(filepath.Join(src, fname))
		}
		if err == nil {
			j.Logger.Debugf("Writing tar body for %q", fname)
			_, err = io.Copy(tw, srcFile)
		}
		if err != nil {
			f.Cancel()
			return fmt.Errorf("couldn't write %q to tar: %s", fname, err)
		}
	}

	j.Logger.Debugf("Closing tar stream")
	err = tw.Close()
	if err != nil {
		f.Cancel()
		return fmt.Errorf("couldn't close tarfile: %s", err)
	}

	j.Logger.Debugf("Moving tarfile from temp file to final location %q", dst)
	err = f.Close()
	if err != nil {
		return fmt.Errorf("couldn't move temp tarfile to %q: %s", dst, err)
	}

	return nil
}
