// issues_queueing.go has the functions and data necessary for sending an issue
// to be processed in the background, and maintaining a list of the issues
// currently being processed so they don't show up in the lists even if their
// files are still on the filesystem.

package sftphandler

import (
	"fileutil"
	"log"
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
	_issuesInProcess[i.Key()] = true
	iipm.Unlock()
	sftpSearcher.ForceReload()

	go startPDFWorkflow(i, workflowPath)
}

// startPDFWorkflow moves the issue out of the SFTP issue location into our "in
// process" bucket, records the issue workflow information in the database, and
// cleans the issue key out of the in-process map.  At this time, it's expected
// that an external cron job will process the PDFs.
func startPDFWorkflow(i *Issue, workflowPath string) {
	// Verify new path will work
	var newLocation = filepath.Join(workflowPath, i.Key())
	if !fileutil.DoesNotExist(newLocation) {
		log.Printf("ERROR - %q already exists; cannot queue issue", newLocation)
		return
	}

	// Move the issue directory to the workflow path
	os.MkdirAll(filepath.Dir(newLocation), 0700)
	log.Println("INFO - Queueing %q to %q", i.Location, newLocation)
	var err = fileutil.CopyDirectory(i.Location, newLocation)
	if err != nil {
		log.Printf("ERROR - unable to copy directory; cannot queue issue: %s", err)
		return
	}
	os.RemoveAll(i.Location)
	i.Location = newLocation

	// TODO: Record the workflow info in the database for external processors
	log.Printf("*** TODO: Record worklow in db ***")

	// Reload the sftp issue list and remove the issue key from the
	// "issuesInProcess" map
	sftpSearcher.ForceReload()
	iipm.Lock()
	delete(_issuesInProcess, i.Key())
	iipm.Unlock()
}

// isIssueInProcess tells the caller if the given issue is being processed so
// it knows not to show it in the UI templates
func isIssueInProcess(key string) bool {
	iipm.Lock()
	var result = _issuesInProcess[key]
	iipm.Unlock()

	return result
}
