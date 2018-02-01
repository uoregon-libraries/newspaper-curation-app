// Package issuewatcher wraps the issuefinder.Finder with some app-specific know-how in order to
// layer on top of the generic issuefinder to include behaviors necessary for
// finding issues from all known locations by reading our settings file and
// running the appropriate searches.
package issuewatcher

import (
	"config"
	"fmt"

	"issuefinder"
	"issuesearch"

	"os"
	"schema"
	"strings"
	"sync"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
)

// A Watcher wraps the local Finder to provide a long-running issue watcher
// which scans issue directories and the live site at regular intervals
type Watcher struct {
	sync.RWMutex
	finder          *issuefinder.Finder
	webroot         string
	tempdir         string
	scanUpload      string
	pdfUpload       string
	lookup          *issuesearch.Lookup
	canonIssues     map[string]*schema.Issue
	status          watcherStatus
	lastFullRefresh time.Time
	done            chan bool
}

type watcherStatus int

const (
	running watcherStatus = 1 << iota
	refreshing
	finished
)

func (ws watcherStatus) String() string {
	var str []string
	if ws&running != 0 {
		str = append(str, "running")
	}
	if ws&refreshing != 0 {
		str = append(str, "refreshing")
	}
	if ws&finished != 0 {
		str = append(str, "finished")
	}
	return strings.Join(str, "/")
}

// New creates an issue Watcher.  Watch() must be called to begin
// looking for issues.
func New(conf *config.Config) *Watcher {
	// We want our first load to reuse the existing cache if available, because
	// an app restart usually happens very shortly after a crash / server reboot
	return &Watcher{
		finder:          issuefinder.New(),
		webroot:         conf.NewsWebroot,
		tempdir:         conf.IssueCachePath,
		scanUpload:      conf.MasterScanUploadPath,
		pdfUpload:       conf.MasterPDFUploadPath,
		canonIssues:     make(map[string]*schema.Issue),
		lastFullRefresh: time.Now(),
		done:            make(chan bool),
	}
}

// IssueFinder returns the underlying issuefinder.Finder.  This will be nil
// until the initial scan is complete
func (w *Watcher) IssueFinder() *issuefinder.Finder {
	return w.finder
}

// Watch loops forever, refreshing the data in the underlying Finder every so
// often.  The refreshing happens on a new issuefinder.Finder which then
// replaces the current finder data, preventing slow searches from holding up
// read access.
func (w *Watcher) Watch(interval time.Duration) {
	w.Lock()

	// If a cache file is available, use it, but we'll still be refreshing data
	// immediately; this just gets the watcher up and running more quickly
	var err = w.Deserialize()
	if err != nil {
		logger.Fatalf("Unable to deserialize the cache file %#v: %s", w.CacheFile(), err)
	}

	if w.status&running != 0 {
		logger.Warnf("Trying to watch issues on an in-progress finder (status: %s)", w.status)
		w.Unlock()
		return
	}
	w.status |= running
	w.Unlock()

	var lastRefresh time.Time
	for {
		if time.Since(lastRefresh) > interval {
			w.refresh()
			lastRefresh = time.Now()
			var err = w.Serialize()
			if err != nil {
				logger.Warnf("Unable to cache to %#v: %s", w.CacheFile(), err)
			}
		}
		time.Sleep(time.Second * 1)

		// If loop should no longer be running, we send the done signal and exit
		w.RLock()
		var stopped = (w.status&running == 0)
		w.RUnlock()
		if stopped {
			w.done <- true
			return
		}
	}
}

// Stop signals the watch loop to stop running, allowing for cleanup to happen safely
func (w *Watcher) Stop() {
	w.Lock()
	if w.status&running == 0 {
		w.Unlock()
		return
	}
	w.status &= ^running
	w.Unlock()

	// Wait for the signal that it's done, then clean up
	w.Lock()
	_ = <-w.done
	w.status = finished
	w.cleanupTempDir()
	w.Unlock()
}

// cleanupTempDir removes the httpcache temp dir files and subdirectories.
// This must have a lock to be used safely.
func (w *Watcher) cleanupTempDir() {
	if w.tempdir == "" {
		return
	}
	var err = os.RemoveAll(w.tempdir)
	if err != nil {
		logger.Errorf("Unable to remove issuewatcher.Watcher's temp dir %#v: %s", w.tempdir, err)
	}
	w.tempdir = ""
}

// makeTempDir creates the temporary directory for httpcache to use.  This does
// nothing if a temporary directory already exists.
func (w *Watcher) makeTempDir() {
	var err = os.MkdirAll(w.tempdir, 0700)
	if err != nil {
		logger.Errorf("Unable to create issuewatcher.Watcher's temp dir: %s", err)
	}
}

// refresh runs the searchers and replaces the underlying issuefinder.Finder.
// Every week, it forces a full refresh of web data as well.
func (w *Watcher) refresh() {
	logger.Debugf("Refreshing issue data")
	w.Lock()

	// Safety: is run off already?  This can only happen if stop was called just
	// as this was about to be called, but it's still better to be safe.
	if w.status&running == 0 {
		logger.Errorf("Trying to refresh a stopped issuewatcher")
		w.Unlock()
		return
	}

	// Don't try to run multiple refreshes!
	if w.status&refreshing != 0 {
		logger.Errorf("Trying to double-refresh a issuewatcher")
		w.Unlock()
		return
	}

	w.status |= refreshing
	w.Unlock()

	// Every week, we force a full web refresh
	var tempdir string
	if time.Since(w.lastFullRefresh) > time.Hour*24*7 {
		logger.Debugf("Purging cache and reindexing all data from scratch")

		// We don't want to delete tempdir when it's a routine cleaning!  TODO: make this way less hacky
		tempdir = w.tempdir
		w.cleanupTempDir()
		w.tempdir = tempdir
		w.lastFullRefresh = time.Now()
	}

	// This won't do anything if we already have a temp dir
	w.makeTempDir()

	// Now actually run the finder and replace it; during this process it's safe
	// for other stuff to happen
	var finder, err = w.FindIssues()

	// This is supposed to happen in the background, so an error can only be
	// reported; we can't do much else....
	if err != nil {
		w.Lock()
		w.status &= ^refreshing
		w.Unlock()
		logger.Errorf("Unable to refresh issuewatcher: %s", err)
		return
	}

	// Create a new lookup using the new finder's data
	var lookup = issuesearch.NewLookup()
	lookup.Populate(finder.Issues)

	// Re-acquire lock to swap out the finder and lookup, then update status
	w.Lock()
	w.finder = finder
	w.lookup = lookup
	w.status &= ^refreshing
	w.Unlock()

	logger.Debugf("Issue data refreshed")
}

// LookupIssues returns a list of schema Issues for the give search key
func (w *Watcher) LookupIssues(key *issuesearch.Key) []*schema.Issue {
	return w.lookup.Issues(key)
}

// IssueErrors returns errors associated with the given issue
func (w *Watcher) IssueErrors(i *schema.Issue) []*issuefinder.Error {
	return w.finder.Errors.IssueErrors[i]
}

// duplicateIssueError returns an error that describes the duplication
func duplicateIssueError(canonical *schema.Issue) error {
	switch canonical.WorkflowStep {
	case schema.WSInProduction:
		return fmt.Errorf("duplicates a live issue in the batch %q", canonical.Batch.Fullname())

	case schema.WSReadyForBatching:
		return fmt.Errorf("duplicates an issue currently being prepped for batching")
	}

	return fmt.Errorf("duplicates an existing issue")
}

// FindIssues calls all the individual find* functions for the myriad of ways
// we store issue information in the various locations, returning the
// issuefinder.Finder with all this data.  Since this operates independently,
// creating and returning a new issuefinder, it's threadsafe and can be called
// with or without the issue watcher loop.
func (w *Watcher) FindIssues() (*issuefinder.Finder, error) {
	var f = issuefinder.New()
	var err error
	var s *issuefinder.Searcher
	var canonIssues = make(map[string]*schema.Issue)

	// Web issues are first as they are live, and therefore always canonical.
	// All issues anywhere else in the workflow that duplicate one of these is
	// unquestionably an error
	s, err = f.FindWebBatches(w.webroot, w.tempdir)
	if err != nil {
		return nil, fmt.Errorf("unable to cache web batches: %s", err)
	}
	for _, issue := range s.Issues {
		canonIssues[issue.Key()] = issue
	}

	// In-process issues are trickier - we label those which are
	// WSReadyForBatching as canonical, *unless* they're a dupe of a live issue,
	// in which case they have to be given an error
	s, err = f.FindInProcessIssues()
	if err != nil {
		return nil, fmt.Errorf("unable to cache in-process issues: %s", err)
	}
	for _, issue := range s.Issues {
		// Check for dupes first
		var k = issue.Key()
		var ci = canonIssues[k]
		if ci != nil {
			s.AddIssueError(issue, duplicateIssueError(ci))
			continue
		}

		// If no dupe, we mark canonical if the issue is ready for batching
		if issue.WorkflowStep == schema.WSReadyForBatching {
			canonIssues[k] = issue
		}
	}

	// SFTP and scanned issues get errors if they're a dupe of anything we've
	// labeled canonical to this point
	s, err = f.FindSFTPIssues(w.pdfUpload)
	if err != nil {
		return nil, fmt.Errorf("unable to cache sftp issues: %s", err)
	}
	for _, issue := range s.Issues {
		var k = issue.Key()
		var ci = canonIssues[k]
		if ci != nil {
			s.AddIssueError(issue, duplicateIssueError(ci))
		}
	}

	s, err = f.FindScannedIssues(w.scanUpload)
	if err != nil {
		return nil, fmt.Errorf("unable to cache scanned issues: %s", err)
	}
	for _, issue := range s.Issues {
		var k = issue.Key()
		var ci = canonIssues[k]
		if ci != nil {
			s.AddIssueError(issue, duplicateIssueError(ci))
		}
	}

	f.Errors.Index()

	return f, nil
}
