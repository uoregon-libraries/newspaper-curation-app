package uploadedissuehandler

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
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
		return fmt.Errorf("unable to build in-process issue list: %w", err)
	}

	var nextScanner = issuewatcher.NewScanner(s.conf).DisableDB().DisableWeb()
	err = nextScanner.Scan()
	if err != nil {
		return fmt.Errorf("unable to scan filesystem: %w", err)
	}

	s.Lock()
	s.lastLoaded = time.Now()
	s.scanner = nextScanner
	s.fails = 0
	s.Unlock()

	s.decorateTitles()

	return nil
}

// decorateTitles iterates over the list of the searcher's titles and decorates
// each, then its issues, and the issues' files, to prepare for web display
func (s *Searcher) decorateTitles() {
	var nextTitles = make([]*Title, 0)
	var nextTitleLookup = make(map[string]*Title)
	for _, t := range s.scanner.Finder.Titles {
		var title, err = s.makeTitle(t)
		if err != nil {
			logger.Errorf("Unable to build title: %s", err)
			continue
		}
		nextTitles = append(nextTitles, title)
		nextTitleLookup[title.Slug()] = title
	}

	s.swapTitleData(nextTitles, nextTitleLookup)
}

// wrapTitle figures out the extra metadata to apply to a title based on its
// location, and returns it.  If the location data can't be parsed properly, an
// error is returned.
func (s *Searcher) wrapTitle(t *schema.Title) (*Title, error) {
	var title = &Title{Title: t}

	// Location is the only element that actually uniquely identifies a title, so
	// we have to use that to figure out if this is a scanned issue or not
	if strings.HasPrefix(t.Location, s.conf.PDFUploadPath) {
		title.Type = TitleTypeBornDigital
		title.MOC = s.conf.PDFBatchMARCOrgCode
		return title, nil
	}

	if strings.HasPrefix(t.Location, s.conf.ScanUploadPath) {
		var relLoc = strings.Replace(title.Location, s.conf.ScanUploadPath, "", 1)
		if relLoc[0] == '/' {
			relLoc = relLoc[1:]
		}
		var parts = strings.Split(relLoc, "/")
		if len(parts) < 2 {
			return nil, fmt.Errorf("bad title location: %q", t.Location)
		}

		title.Type = TitleTypeScanned
		title.MOC = parts[0]
		return title, nil
	}

	return nil, fmt.Errorf("unknown title location: %q", t.Location)
}

func (s *Searcher) makeTitle(t *schema.Title) (*Title, error) {
	var title, err = s.wrapTitle(t)
	if err != nil {
		return nil, err
	}

	title.decorateIssues(t.Issues)
	return title, nil
}

func (s *Searcher) swapTitleData(nextTitles []*Title, nextTitleLookup map[string]*Title) {
	s.Lock()
	defer s.Unlock()

	s.titles = nextTitles
	s.titleLookup = nextTitleLookup

	// We like titles sorted by name for presentation
	sort.Slice(s.titles, func(i, j int) bool {
		return strings.ToLower(s.titles[i].Name) < strings.ToLower(s.titles[j].Name)
	})
}

// BuildInProcessList pulls all pending SFTP move jobs from the database and
// indexes them by location in order to avoid showing issues which are already
// awaiting processing.
func (s *Searcher) BuildInProcessList() error {
	var nextInProcessIssues = make(map[string]bool)

	var issues, err = models.Issues().InWorkflowStep(schema.WSAwaitingProcessing).Fetch()
	if err != nil {
		return fmt.Errorf("unable to find in-process issues: %w", err)
	}

	for _, issue := range issues {
		nextInProcessIssues[issue.Key()] = true
	}

	s.Lock()
	s.inProcessIssues = nextInProcessIssues
	s.Unlock()

	return nil
}

// RemoveIssue takes the given issue out of all lookups to hide it from
// front-end queueing operations
func (s *Searcher) RemoveIssue(i *Issue) {
	s.Lock()
	defer s.Unlock()

	s.inProcessIssues[i.Key()] = true

	var newIssues []*Issue
	for _, issue := range i.Title.Issues {
		if issue != i {
			newIssues = append(newIssues, issue)
		}
	}

	i.Title.Issues = newIssues
	delete(i.Title.IssueLookup, i.Slug)
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
func (s *Searcher) TopErrors() apperr.List {
	s.RLock()
	defer s.RUnlock()
	return s.scanner.Finder.Errors
}
