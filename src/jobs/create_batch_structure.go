package jobs

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// CreateBatchStructure wraps a BatchJob and implements Processor
type CreateBatchStructure struct {
	*BatchJob
}

// Process implements Processor by creating the batch directory structure and
// hard-linking the necessary issue files
func (j *CreateBatchStructure) Process(*config.Config) ProcessResponse {
	var err error

	// Configure the paths
	var wipPath = j.db.Args[JobArgLocation]
	if !fileutil.MustNotExist(wipPath) {
		j.Logger.Errorf("Directory %q already exists", wipPath)
		return PRFatal
	}
	var dataPath = path.Join(wipPath, "data")

	// Create the top-level directories
	err = os.MkdirAll(dataPath, 0755)
	if err != nil {
		j.Logger.Criticalf("Unable to create WIP data directory %q: %s", dataPath, err)
		return PRFailure
	}

	// Iterate over issues to generate the issue directories and hard-link all
	// the files
	var iList []*models.Issue
	iList, err = j.DBBatch.Issues()
	if err != nil {
		j.Logger.Criticalf("Unable to read issues for %q: %s", j.DBBatch.FullName(), err)
		return PRFailure
	}
	for _, issue := range iList {
		var destPath = path.Join(dataPath, issue.LCCN, "print", issue.DateEdition())
		err = os.MkdirAll(destPath, 0755)
		if err == nil {
			err = linkFiles(issue.Location, destPath)
		}
		if err != nil {
			j.Logger.Criticalf("Unable to link issue %q into batch %q: %s", issue.Key(), j.DBBatch.FullName(), err)
			return PRFailure
		}
	}

	// Success!
	return PRSuccess
}

// linkFiles hard-links all regular, non-hidden files from src into dest
func linkFiles(src string, dest string) error {
	var entries, err = os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("couldn't scan for source files: %w", err)
	}

	for _, entry := range entries {
		var info, err = entry.Info()
		if err != nil {
			return fmt.Errorf("couldn't get file detail for %q: %w", entry.Name(), err)
		}

		if !info.Mode().IsRegular() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		var name = entry.Name()
		var srcPath = filepath.Join(src, name)
		var destPath = filepath.Join(dest, name)
		err = os.Link(srcPath, destPath)
		if err != nil {
			return fmt.Errorf("couldn't link %q to %q: %w", srcPath, destPath, err)
		}
	}
	return nil
}
