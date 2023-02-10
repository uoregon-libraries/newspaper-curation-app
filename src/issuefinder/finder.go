// Package issuefinder sets up a process for finding all issues across the
// filesystem and live sites to allow for other tools to get fairly
// comprehensive information: where in the workflow an issue resides,
// which batches contain a certain LCCN, which issues have dupes, etc.
package issuefinder

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// Namespace is a special type of identifier for searchers to have well-defined
// namespacing for the different types of searches / locations.  If two
// different locations need the same namespace, a different top-level Finder
// should be used (e.g., finding "web" issues on live versus staging)
type Namespace uint8

// These are the allowed namespaces for searcher, based on our current app's
// workflow location types
const (
	Website Namespace = iota
	SFTPUpload
	ScanUpload
	InProcess
)

// Searcher is the central component of the issuefinder package, running the filesystem
// and web queries and providing an API to get the results
type Searcher struct {
	Namespace Namespace
	Location  string

	Issues  schema.IssueList
	Batches []*schema.Batch
	Titles  schema.TitleList

	// dbTitles holds a temporary cache (living for the life of this Searcher) of
	// all titles in the database
	dbTitles models.TitleList

	// titleByLoc holds titles keyed by their location so we don't duplicate the
	// same title entry if it's in the same place.  This is most applicable to
	// live titles, since they're unique per LCCN.
	titleByLoc map[string]*schema.Title

	// Errors is the list of errors which aren't specific to something like an
	// issue or a batch; e.g., a bad MARC Org Code directory in the scan path
	Errors apperr.List
}

// Finder groups all the searchers together, allowing for aggregation of issue,
// title, and batch data from all sources while keeping the groups separate for
// the specific use-cases (e.g., SFTP issues shouldn't be scanned when trying
// to figure out if a given issue is live).  A Finder doesn't have any critical
// context on its own, and can be reproduced from data stored in its Searchers.
type Finder struct {
	Searchers map[Namespace]*Searcher
	Batches   []*schema.Batch
	Titles    schema.TitleList
	Issues    schema.IssueList
	Errors    apperr.List

	// This little var helps us answer the age-old question: for a given unique
	// issue, where is it in the workflow?
	IssueNamespace map[*schema.Issue]Namespace
}

// New instantiates a new Finder read to spawn searchers
func New() *Finder {
	return &Finder{
		Searchers:      make(map[Namespace]*Searcher),
		IssueNamespace: make(map[*schema.Issue]Namespace),
	}
}

// NewSearcher instantiates a Searcher on its own, and typically isn't needed,
// but could be useful for specific one-off scripts
func NewSearcher(ns Namespace, loc string) (*Searcher, error) {
	var s = &Searcher{Namespace: ns, Location: loc}
	var err = s.init()
	return s, err
}

func (s *Searcher) init() error {
	s.Issues = make(schema.IssueList, 0)
	s.Batches = make([]*schema.Batch, 0)
	s.Titles = make(schema.TitleList, 0)
	s.titleByLoc = make(map[string]*schema.Title)
	s.Errors.Clear()

	// Make sure titles are loaded from the DB, and returns errors
	var err error
	s.dbTitles, err = models.Titles()
	if err != nil {
		return fmt.Errorf("reading titles from database for new Searcher: %s", err)
	}

	return nil
}

func (f *Finder) storeSearcher(s *Searcher) {
	f.Searchers[s.Namespace] = s
}

// createAndProcessSearcher instantiates a new Searcher, passes it to
// Processor, aggregates its data in the Finder, and returns the error, if any
func (f *Finder) createAndProcessSearcher(ns Namespace, loc string, processor func(s *Searcher) error) (*Searcher, error) {
	var s, err = NewSearcher(ns, loc)
	if err == nil {
		err = processor(s)
	}
	if err != nil {
		return nil, err
	}

	f.storeSearcher(s)
	f.aggregate(s)

	return s, nil
}

// FindSFTPIssues creates and runs an SFTP Searcher, aggregates its data,
// and returns any errors encountered
func (f *Finder) FindSFTPIssues(path, orgCode string) (*Searcher, error) {
	var searchFn = func(s *Searcher) error { return s.FindSFTPIssues(orgCode) }
	return f.createAndProcessSearcher(SFTPUpload, path, searchFn)
}

// FindScannedIssues creates and runs a scanned-issue Searcher, aggregates its
// data, and returns any errors encountered
func (f *Finder) FindScannedIssues(path string) (*Searcher, error) {
	var searchFn = func(s *Searcher) error { return s.FindScannedIssues() }
	return f.createAndProcessSearcher(ScanUpload, path, searchFn)
}

// FindWebBatches creates and runs a website batch Searcher, aggregates its
// data, and returns any errors encountered
func (f *Finder) FindWebBatches(hostname, cachePath string) (*Searcher, error) {
	var searchFn = func(s *Searcher) error { return s.FindWebBatches(cachePath) }
	return f.createAndProcessSearcher(Website, hostname, searchFn)
}

// FindInProcessIssues creates and runs an in-process issues (issues which are
// in the workflow dir and have been indexed) searcher, aggregates its data,
// and returns any errors encountered
func (f *Finder) FindInProcessIssues() (*Searcher, error) {
	var searchFn = func(s *Searcher) error { return s.FindInProcessIssues() }
	return f.createAndProcessSearcher(InProcess, "database", searchFn)
}

// aggregate just puts the searcher's data into the Finder for global use
func (f *Finder) aggregate(s *Searcher) {
	for _, b := range s.Batches {
		f.Batches = append(f.Batches, b)
	}
	for _, t := range s.Titles {
		f.Titles = append(f.Titles, t)
	}
	for _, i := range s.Issues {
		f.Issues = append(f.Issues, i)
		f.IssueNamespace[i] = s.Namespace
	}
	for _, e := range s.Errors.All() {
		f.Errors.Append(e)
	}
}

// Aggregate puts all searchers' data into the Finder for global use.  This
// must be called if batches, issues, titles, or errors are added to a searcher
// directly (rather than via FindXXX methods).
func (f *Finder) Aggregate() {
	f.Batches = nil
	f.Titles = nil
	f.Issues = nil
	f.IssueNamespace = make(map[*schema.Issue]Namespace)
	f.Errors.Clear()

	for _, s := range f.Searchers {
		f.aggregate(s)
	}
}
