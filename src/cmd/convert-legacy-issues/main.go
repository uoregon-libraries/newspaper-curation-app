// This utility is very specific to a (please oh please be true) one-off
// conversion to get our old filesystem-based issues converted.  It is not
// likely to be useful to others outside the UO.  Hopefully I remember to
// remove it.  But I probably won't.

package main

import (
	"cli"
	"config"
	"db"
	"jobs"
	"path/filepath"
	"schema"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
)

// Command-line options
type _opts struct {
	cli.BaseOptions
	DryRun bool `short:"n" long:"dry-run" description:"don't write to the database"`
}

var (
	opts _opts
	conf *config.Config
)

func getConf() {
	var c = cli.New(&opts)
	conf = c.GetConf()

	// Just to be extra-safe, let's not even setup the database connection for a
	// dry run
	if !opts.DryRun {
		var err = db.Connect(conf.DatabaseConnect)
		if err != nil {
			logger.Fatalf("Error trying to connect to database: %s", err)
		}
	}
}

func processIssue(path string) {
	var alreadyInDB bool

	var dbi, err = getDBIssue(path)
	if err != nil {
		logger.Errorf("Error in issue dir %q: %s", path, err)
		return
	}

	var info = "new"
	if dbi.ID > 0 {
		alreadyInDB = true
		info = "existing"
	}

	logger.Debugf("Processing %q (%s)", path, info)

	// Try to find a master backup for this issue - I do this by directory name
	// because of an out-of-band rename job I did to try and reorganize stuff.
	// If you aren't me, this won't work for you.
	var base = filepath.Base(path)
	var backupPath = filepath.Join(conf.MasterPDFBackupPath, base)
	if fileutil.IsDir(backupPath) {
		logger.Debugf("Found master backup at %q", backupPath)
		dbi.MasterBackupLocation = backupPath
	}

	if opts.DryRun {
		logger.Debugf("Dry run - not saving to database")
	} else {
		logger.Debugf("Saving issue to database")
		err = dbi.Save()
		if err != nil {
			logger.Errorf("Error trying to save issue from dir %q: %s", path, err)
			return
		}
	}

	// Don't queue jobs if the issue was already in the database - chances are
	// the jobs were already created
	if !alreadyInDB {
		jobs.QueueSerial(
			jobs.PrepareIssueJobAdvanced(jobs.JobTypeMoveIssueToWorkflow, dbi, path, schema.WSNil),
			jobs.PrepareIssueJobAdvanced(jobs.JobTypeMakeDerivatives, dbi, path, schema.WSAwaitingMetadataReview),
		)
	}
}

func getIncomingIssuePaths() (paths []string) {
	var newPath = filepath.Join(conf.WorkflowPath, "..", "redos", "new-workflow", "incoming")
	var issueDirs, err = fileutil.FindDirectories(newPath)
	if err != nil {
		logger.Fatalf("Unable to scan for issues in %q: %s", newPath, err)
	}

	for _, dir := range issueDirs {
		logger.Debugf("Scanning issue dir %q", dir)
		paths = append(paths, dir)
	}

	return paths
}

func main() {
	getConf()
	var issuePaths = getIncomingIssuePaths()
	for _, path := range issuePaths {
		processIssue(path)
	}
}
