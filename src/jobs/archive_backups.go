package jobs

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
)

// ArchiveBackups is the job which finds backed-up original files (if any
// exist), creates an archive from them, and copies them into the issue
// location.  This only matters to born-digital issues; the scanned issues
// aren't pre-processed like PDFs are.
type ArchiveBackups struct {
	*IssueJob
	tarfile string
}

// Process implements Processor, moving the issue's original files
func (j *ArchiveBackups) Process(*config.Config) ProcessResponse {
	if j.DBIssue.BackupLocation == "" {
		j.Logger.Debugf("Archive job for issue id %d skipped - no backup exists", j.DBIssue.ID)
		return PRSuccess
	}

	j.tarfile = filepath.Join(j.DBIssue.Location, "original.tar")
	j.Logger.Debugf("Creating archive for issue id %d", j.DBIssue.ID)
	var err = j.buildArchive()
	if err != nil {
		j.Logger.Errorf("Unable to produce tarfile from PDF(s): %s", err)
		return PRFailure
	}

	// Verify the tar wrote successfully just to be uber-paranoid before we go
	// deleting the original file(s)
	var info os.FileInfo
	info, err = os.Stat(j.tarfile)
	if err != nil {
		j.Logger.Errorf("Unable to stat tarfile: %s", err)
		return PRFailure
	}
	if info.Size() == 0 {
		j.Logger.Errorf("Generated tarfile is 0 bytes")
		return PRFailure
	}

	return PRSuccess
}

func (j *ArchiveBackups) buildArchive() error {
	var src = j.DBIssue.BackupLocation

	var f = fileutil.NewSafeFile(j.tarfile)
	defer f.Close()

	var tw = tar.NewWriter(f)
	var entries, err = os.ReadDir(src)
	if err != nil {
		f.Cancel()
		return fmt.Errorf("couldn't read %q: %w", src, err)
	}

	for _, entry := range entries {
		var fname = entry.Name()
		var info fs.FileInfo
		info, err = entry.Info()
		if err != nil {
			f.Cancel()
			return fmt.Errorf("couldn't get size of %q: %w", entry.Name(), err)
		}
		j.Logger.Debugf("Writing tar header for %q", fname)
		var hdr = &tar.Header{
			Name: filepath.Join("original", fname),
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
			return fmt.Errorf("couldn't write %q to tar: %w", fname, err)
		}
	}

	j.Logger.Debugf("Closing tar stream")
	err = tw.Close()
	if err != nil {
		f.Cancel()
		return fmt.Errorf("couldn't close tarfile: %w", err)
	}

	j.Logger.Debugf("Moving tarfile from temp file to final location %q", j.tarfile)
	err = f.Close()
	if err != nil {
		return fmt.Errorf("couldn't move temp tarfile to %q: %w", j.tarfile, err)
	}

	return nil
}
