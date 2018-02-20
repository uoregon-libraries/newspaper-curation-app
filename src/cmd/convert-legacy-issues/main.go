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

var conf *config.Config

func getConf() {
	var c = cli.Simple()
	conf = c.GetConf()
	var err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}
}

func processIssue(path string) {
	var dbi, err = getDBIssue(path)
	if err != nil {
		logger.Errorf("Error in issue dir %q: %s", path, err)
		return
	}

	// TODO: Try to find a master backup for this issue
	// dbi.MasterBackupLocation

	err = dbi.Save()
	if err != nil {
		logger.Errorf("Error trying to save issue from dir %q: %s", path, err)
		return
	}

	jobs.QueueSerial(
		jobs.PrepareIssueJobAdvanced(jobs.JobTypeMoveIssueToWorkflow, dbi, path, schema.WSNil),
		jobs.PrepareIssueJobAdvanced(jobs.JobTypeMakeDerivatives, dbi, path, schema.WSAwaitingMetadataReview),
	)
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
