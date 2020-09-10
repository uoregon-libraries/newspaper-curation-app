package jobs

import (
	"archive/tar"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
)

// ArchiveBackups is the job which finds backed-up master files (if any
// exist), creates an archive from them, and copies them into the issue
// location.  This only matters to born-digital issues; the scanned issues
// aren't pre-processed like PDFs are.
type ArchiveBackups struct {
	*IssueJob
	tarfile string
}

// Process implements Processor, moving the issue's master files
func (j *ArchiveBackups) Process(*config.Config) bool {
	if j.DBIssue.BackupLocation == "" {
		j.Logger.Debugf("Master file archive job for issue id %d skipped - no master files stored", j.DBIssue.ID)
		return true
	}

	j.tarfile = filepath.Join(j.DBIssue.Location, "master.tar")
	j.Logger.Debugf("Starting master file archive for issue id %d", j.DBIssue.ID)
	var err = j.makeMasterTar()
	if err != nil {
		j.Logger.Errorf("Unable to produce tarfile from master PDF(s): %s", err)
		return false
	}

	// Verify the tar wrote successfully just to be uber-paranoid before we go
	// deleting the original master
	var info os.FileInfo
	info, err = os.Stat(j.tarfile)
	if err != nil {
		j.Logger.Errorf("Unable to stat tarfile: %s", err)
		return false
	}
	if info.Size() == 0 {
		j.Logger.Errorf("Generated tarfile is 0 bytes")
		return false
	}

	return true
}

func (j *ArchiveBackups) makeMasterTar() error {
	var src = j.DBIssue.BackupLocation

	var f = fileutil.NewSafeFile(j.tarfile)
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

	j.Logger.Debugf("Moving tarfile from temp file to final location %q", j.tarfile)
	err = f.Close()
	if err != nil {
		return fmt.Errorf("couldn't move temp tarfile to %q: %s", j.tarfile, err)
	}

	return nil
}
