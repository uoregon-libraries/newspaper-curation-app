package uploadedissuehandler

import (
	"config"
	"db"
	"fmt"
	"issuefinder"
	"issuewatcher"
	"jobs"
	"schema"
	"sync"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
)

// secondsBetweenIssueReload should be a value that ensures nearly real-time
// data, but avoids hammering the disk if a lot of refreshing happens
const secondsBetweenIssueReload = 60

// maxLoadFailures is the number of times in a row a scan may fail before we
// consider the system in a failed state and respond to requests with an error
const maxLoadFailures = 5

// Searcher holds onto a Scanner for running local queries against scan
// and sftp uploads.  This structure is completely thread-safe; a single
// instance can and should used for the life of the web server.  All data
// access is via functions to allow automatic rescanning of the file system.
type Searcher struct {
	sync.RWMutex
	conf            *config.Config
	lastLoaded      time.Time
	scanner         *issuewatcher.Scanner
	titles          []*Title
	titleLookup     map[string]*Title
	inProcessIssues map[string]bool
	fails           int
}

// newSearcher returns a searcher that wraps issuefinder and schema data for
// web presentation of titles, issues, files, and errors in SFTP/scanned
// uploads
func newSearcher(conf *config.Config) *Searcher {
	var s = &Searcher{conf: conf}
	go s.watch()
	return s
}

// watch checks the time since the last load, and loads issues from the
// filesystem if necessary.  If issues were loaded, the various types are
// decorated as needed for web presentation.  This should be run in a goroutine
// as it loops forever.
func (s *Searcher) watch() {
	for {
		s.RLock()
		var since = time.Since(s.lastLoaded)
		s.RUnlock()

		if since >= time.Second*secondsBetweenIssueReload {
			var err = s.scan()
			if err != nil {
				s.Lock()
				s.fails++
				s.Unlock()
				logger.Errorf("Searcher.scan(): %s", err)
			}
		}

		time.Sleep(time.Second)
	}
}

func (s *Searcher) scan() error {
	var err = s.BuildInProcessList()
	if err != nil {
		return fmt.Errorf("unable to build in-process issue list: %s", err)
	}

	var nextScanner = issuewatcher.NewScanner(s.conf).DisableDB().DisableWeb()
	err = nextScanner.Scan()
	if err != nil {
		return fmt.Errorf("unable to scan filesystem: %s", err)
	}

	s.Lock()
	s.lastLoaded = time.Now()
	s.scanner = nextScanner
	s.fails = 0
	s.Unlock()

	s.decorateTitles()

	return nil
}

// BuildInProcessList pulls all pending SFTP move jobs from the database and
// indexes them by location in order to avoid showing issues which are already
// awaiting processing.
func (s *Searcher) BuildInProcessList() error {
	var nextInProcessIssues = make(map[string]bool)

	var jobs, err = db.FindRecentJobsByType(string(jobs.JobTypeMoveIssueToWorkflow), time.Second*secondsBetweenIssueReload)
	if err != nil {
		return fmt.Errorf("unable to find recent issue move jobs: %s", err)
	}

	for _, job := range jobs {
		var dbi, err = db.FindIssue(job.ObjectID)
		if err != nil {
			return fmt.Errorf("unable to get issue for job id %d: %s", job.ID, err)
		}
		if dbi == nil {
			return fmt.Errorf("no issue with id %d exists", job.ObjectID)
		}

		var si *schema.Issue
		si, err = dbi.SchemaIssue()
		if err != nil {
			return err
		}
		nextInProcessIssues[si.Key()] = true
	}

	s.Lock()
	s.inProcessIssues = nextInProcessIssues
	s.Unlock()

	return nil
}

// IsInProcess returns whether the given issue key has been seen in the
// in-process issue list
func (s *Searcher) IsInProcess(issueKey string) bool {
	s.RLock()
	defer s.RUnlock()
	return s.inProcessIssues[issueKey]
}

// Titles returns the list of titles
func (s *Searcher) Titles() []*Title {
	s.RLock()
	defer s.RUnlock()
	return s.titles
}

// FailedSearch returns true if too many scans have failed in a row
func (s *Searcher) FailedSearch() bool {
	s.RLock()
	defer s.RUnlock()
	return s.fails > maxLoadFailures
}

// TitleLookup returns the Title for a given slug
func (s *Searcher) TitleLookup(slug string) *Title {
	s.RLock()
	defer s.RUnlock()
	return s.titleLookup[slug]
}

// Ready returns whether or not the searcher has completed at least one search
func (s *Searcher) Ready() bool {
	s.RLock()
	defer s.RUnlock()
	return s.scanner != nil
}

// TopErrors returns the list of errors found that weren't tied to an issue/title/file
func (s *Searcher) TopErrors() []*issuefinder.Error {
	s.RLock()
	defer s.RUnlock()
	return s.scanner.Finder.Errors.OtherErrors
}
