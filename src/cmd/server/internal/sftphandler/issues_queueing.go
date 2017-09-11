// issues_queueing.go has the functions and data necessary for sending an issue
// to be processed in the background, and maintaining a list of the issues
// currently being processed so they don't show up in the lists even if their
// files are still on the filesystem.

package sftphandler

import (
	"db"
	"fileutil"
	"fmt"
	"logger"
	"os"
	"path/filepath"
	"sync"
)

// _issuesInProcess stores "true" for issue keys that are currently in the
// process of being queued for derivatives
var _issuesInProcess = make(map[string]bool)
var iipm sync.Mutex

func queueIssueForProcessing(i *Issue, workflowPath string) {
	iipm.Lock()
	var alreadyInProcess = _issuesInProcess[i.Key()]
	_issuesInProcess[i.Key()] = true
	iipm.Unlock()
	sftpSearcher.ForceReload()

	if !alreadyInProcess {
		go startPDFWorkflow(i, workflowPath)
	}
}

// startPDFWorkflow moves the issue out of the SFTP issue location into our "in
// process" bucket, records the issue workflow information in the database, and
// cleans the issue key out of the in-process map.  At this time, it's expected
// that an external cron job will process the PDFs.
func startPDFWorkflow(i *Issue, workflowPath string) {
	// Make sure we release the issue key no matter what else happens
	defer func() {
		iipm.Lock()
		delete(_issuesInProcess, i.Key())
		iipm.Unlock()
	}()

	var err error
	var dbi *db.Issue
	var saveOrCrit = func(format string, args ...interface{}) bool {
		var err = dbi.Save()
		if err != nil {
			var fullmsg = fmt.Sprintf(format, args...)
			logger.Critical("%s: %s (issue %q)", fullmsg, err, i.Key())
		}
		return err == nil
	}

	// Check for an existing database issue workflow in case this issue failed
	// queuing previously
	dbi, err = db.FindIssueByKey(i.Key())
	if err != nil {
		dbi.Info = "Unable to connect to the database.  Try again or contact the system administrator."
		saveOrCrit("Couldn't store error information")
		logger.Critical("Couldn't search the database for %q: %s", i.Key(), err)
		return
	}

	if dbi == nil {
		// If we don't have an issue, we need to create one using the sftp
		// structure's data
		dbi = db.NewIssue(i.Location)
		dbi.LCCN = i.Title.LCCN
		dbi.Date = i.DateStringReadable()
		dbi.Edition = i.Edition
		dbi.WorkflowStep = db.WSPreppingSFTPIssueForMove

		// Make sure we record the issue info in the database right away so we can
		// track things if the move operation fails
		if !saveOrCrit("Couldn't create initial workflow data") {
			return
		}
	} else {
		// If we have an issue, its workflow status and location must match this
		// issue, otherwise we've got some kind of wonky dupe.  This can't even be
		// communicated to the person who queued the issue, given that it's an
		// async operation and the database already has a workflow record, so we'll
		// need to take steps to let them know ahead of time.
		var fail bool
		if dbi.Location != i.Location {
			logger.Critical("%q is being tracked at %q; our issue is in %q", i.Key(), dbi.Location, i.Location)
			fail = true
		}
		if dbi.WorkflowStep != db.WSPreppingSFTPIssueForMove {
			logger.Critical("%q is being tracked with an unexpected workflow step, %d", i.Key(), dbi.WorkflowStep)
			fail = true
		}

		if fail {
			// TODO: Figure out a way to store alerts / incidents for a given user so
			// we can alert the user without an issue-specific update.  This right
			// here would actually break an in-process issue that got double-clicked
			// or something weird.  SEE ABOVE COMMENT!!!
			//
			// dbi.Error = "This issue appears to be an untracked dupe.  Contact the system administrator."
			// saveOrCrit("Couldn't store error information")
			return
		}
	}

	// Verify new path will work
	var newLocation = filepath.Join(workflowPath, i.Key())
	if !fileutil.MustNotExist(newLocation) {
		dbi.Info = fmt.Sprintf("The issue was unable to be queued due to the " +
			"destination folder already existing.  You may attempt to queue this " +
			"issue again if there are no other errors, but it may be a duplicate.")
			saveOrCrit("Couldn't store info")
		logger.Error("Unable to queue issue: %q already exists", newLocation)
		return
	}

	// Move the issue directory to the workflow path
	var wipLocation = newLocation + "-wip"
	os.MkdirAll(filepath.Dir(wipLocation), 0700)
	logger.Info("Queueing %q to %q", i.Location, wipLocation)
	err = fileutil.CopyDirectory(i.Location, wipLocation)
	if err != nil {
		dbi.Error = fmt.Sprintf("Unable to move the issue for processing - " +
			"contact the system administrator for help")
		saveOrCrit("Couldn't store error")
		logger.Error("Unable to copy directory; cannot queue issue: %s", err)
		return
	}
	err = os.RemoveAll(i.Location)
	if err != nil {
		dbi.Error = fmt.Sprintf("Error trying to clean up after previous queue " +
			"operation - contact the system administrator for help")
		saveOrCrit("Couldn't store error")
		logger.Error("Unable to remove old issue directory post-copy: %s", err)
		return
	}
	err = os.Rename(wipLocation, newLocation)
	if err != nil {
		dbi.Error = fmt.Sprintf("Error trying to clean up after previous queue " +
			"operation - contact the system administrator for help")
		saveOrCrit("Couldn't store error")
		logger.Error("Unable to rename WIP issue directory (%q -> %q) post-copy: %s", wipLocation, newLocation, err)
		return
	}
	i.Location = newLocation

	// This is tricky - if we can't update the workflow, but the move succeeded,
	// there's not much we can do but log
	dbi.Location = newLocation
	dbi.WorkflowStep = db.WSAwaitingPDFProcessing
	dbi.Error = ""
	dbi.Info = ""
	saveOrCrit("Couldn't update location and workflow step after the move")

	// Forcibly reload the sftp issue list if all went well
	sftpSearcher.ForceReload()
}

// isIssueInProcess tells the caller if the given issue is being processed so
// it knows not to show it in the UI templates
func isIssueInProcess(key string) bool {
	iipm.Lock()
	var result = _issuesInProcess[key]
	iipm.Unlock()

	return result
}
