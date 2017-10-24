package sftphandler

import (
	"db"
	"fmt"
	"issuefinder"
	"jobs"
	"schema"
	"sync"
	"time"
)

// secondsBetweenSFTPReload should be a value that ensures nearly real-time
// data, but avoids hammering the disk if a lot of refreshing happens
const secondsBetweenSFTPReload = 30

// secondsBeforeFatalError is how long we allow the system to run with an error
// response before we actually return a failure from any functions which
// require searching the filesystem
const secondsBeforeFatalError = 600

// SFTPSearcher wraps an issuefinder.Searcher specifically for SFTP data.
// This is thread-safe; a single instance can and should used for the life of
// the web server.  All data access is via functions to allow automatic
// rescanning of the file system.
type SFTPSearcher struct {
	sync.Mutex
	lastLoaded      time.Time
	searcher        *issuefinder.Searcher
	titles          []*Title
	titleLookup     map[string]*Title
	inProcessIssues sync.Map
}

// newSFTPSearcher returns a searcher that wraps issuefinder and schema data
// for web presentation of titles, issues, files, and errors in SFTP uploads
func newSFTPSearcher(path string) *SFTPSearcher {
	return &SFTPSearcher{searcher: issuefinder.NewSearcher(issuefinder.SFTPUpload, path)}
}

// load checks the time since the last load, and loads issues from the
// filesystem if necessary.  If issues were loaded, the various types are
// decorated as needed for web presentation.
func (s *SFTPSearcher) load() error {
	s.Lock()
	defer s.Unlock()

	if time.Since(s.lastLoaded) < time.Second*secondsBetweenSFTPReload {
		return nil
	}

	var err = s.buildInProcessList()
	if err != nil {
		return err
	}

	err = s.searcher.FindSFTPIssues()
	if err == nil {
		s.lastLoaded = time.Now()
		s.decorateTitles()
	}
	return err
}

// buildInProcessList pulls all pending SFTP move jobs from the database and
// indexes them by location in order to avoid showing issues which are already
// awaiting processing.
//
// The searcher must be locked here, as it completely replaces inProcessIssues.
func (s *SFTPSearcher) buildInProcessList() error {
	s.inProcessIssues = sync.Map{}

	var dbJobs, err = db.FindJobsByStatusAndType(string(jobs.JobStatusPending), string(jobs.JobTypeSFTPIssueMove))
	if err != nil {
		return fmt.Errorf("unable to find pending sftp issue move jobs: %s", err)
	}

	for _, dbJob := range dbJobs {
		var dbi, err = db.FindIssue(dbJob.ObjectID)
		if err != nil {
			return fmt.Errorf("unable to get issue for job id %d: %s", dbJob.ID, err)
		}
		if dbi == nil {
			return fmt.Errorf("no issue with id %d exists", dbJob.ObjectID)
		}

		var si *schema.Issue
		si, err = dbi.SchemaIssue()
		if err != nil {
			return err
		}
		s.inProcessIssues.Store(si.Key(), true)
	}

	return nil
}

// Titles returns the list of titles in the SFTP directory
func (s *SFTPSearcher) Titles() ([]*Title, error) {
	var err = s.load()
	if err != nil && time.Since(s.lastLoaded) > secondsBeforeFatalError {
		return nil, err
	}

	return s.titles, nil
}

// ForceReload clears the last loaded time and refreshed the titles cache
func (s *SFTPSearcher) ForceReload() {
	s.lastLoaded = time.Time{}
	s.Titles()
}

// TitleLookup returns the Title for a given LCCN
func (s *SFTPSearcher) TitleLookup(lccn string) *Title {
	s.load()
	return s.titleLookup[lccn]
}
