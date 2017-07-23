// issues_queueing.go has the functions and data necessary for sending an issue
// to be processed in the background, and maintaining a list of the issues
// currently being processed so they don't show up in the lists even if their
// files are still on the filesystem.

package sftphandler

import (
	"log"
	"sync"
	"time"
)

// _issuesInProcess stores "true" for issue keys that are currently in the
// process of being queued for derivatives
var _issuesInProcess = make(map[string]bool)
var iipm sync.Mutex

func queueIssueForDerivatives(i *Issue) {
	iipm.Lock()
	_issuesInProcess[i.Key()] = true
	iipm.Unlock()
	sftpSearcher.ForceReload()

	go processDerivatives(i)
}

// processDerivatives moves the issue out of the SFTP issue location and runs
// the derivative processor
func processDerivatives(i *Issue) {
	// Move the issue directory
	log.Println("Hiding issue for one minute")
	time.Sleep(time.Minute)

	// Remove the issue key from the "issuesInProcess" map
	iipm.Lock()
	delete(_issuesInProcess, i.Key())
	log.Println("Issue no longer hidden")
	iipm.Unlock()

	// Generate derivatives for the issue
}

// isIssueInProcess tells the caller if the given issue is being processed so
// it knows not to show it in the UI templates
func isIssueInProcess(key string) bool {
	iipm.Lock()
	var result = _issuesInProcess[key]
	iipm.Unlock()

	return result
}
