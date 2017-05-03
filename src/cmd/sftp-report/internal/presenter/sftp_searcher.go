package presenter

import (
	"issuefinder"
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
	lastLoaded  time.Time
	searcher    *issuefinder.Searcher
	titles      []*Title
	titleLookup map[string]*Title
}

// NewSFTPSearcher returns a searcher that wraps issuefinder and schema data
// for web presentation of titles, issues, files, and errors in SFTP uploads
func NewSFTPSearcher(path string) *SFTPSearcher {
	return &SFTPSearcher{searcher: issuefinder.NewSearcher(issuefinder.SFTPUpload, path)}
}

// sftpLoad checks the time since the last load, and loads issues from the
// filesystem if necessary.  If issues were loaded, the various types are
// decorated as needed for web presentation.
func (s *SFTPSearcher) load() error {
	if time.Since(s.lastLoaded) < time.Second*secondsBetweenSFTPReload {
		return nil
	}

	var err = s.searcher.FindSFTPIssues()
	if err == nil {
		s.lastLoaded = time.Now()
		s.decorateTitles()
	}
	return err
}

// GetSFTPTitles returns the list of titles in the SFTP directory
func (s *SFTPSearcher) Titles() ([]*Title, error) {
	s.Lock()
	defer s.Unlock()

	var err = s.load()
	if err != nil && time.Since(s.lastLoaded) > secondsBeforeFatalError {
		return nil, err
	}

	return s.titles, nil
}

func (s *SFTPSearcher) TitleLookup(lccn string) *Title {
	s.Lock()
	defer s.Unlock()
	s.load()
	return s.titleLookup[lccn]
}
