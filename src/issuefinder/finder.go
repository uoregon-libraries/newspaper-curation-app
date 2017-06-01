// Package issuefinder sets up a process for finding all issues across the
// filesystem and live sites to allow for other tools to get fairly
// comprehensive information: where in the workflow an issue resides,
// which batches contain a certain LCCN, which issues have dupes, etc.
package issuefinder

import (
	"schema"
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
	AwaitingPageReview
	AwaitingMetadataReview
	PDFsAwaitingDerivatives
	ScansAwaitingDerivatives
	ReadyForBatching
	BatchedOnDisk
	MasterBackup
	PageBackup
)

// Searcher is the central component of the issuefinder package, running the filesystem
// and web queries and providing an API to get the results
type Searcher struct {
	Namespace Namespace
	Location  string

	Issues  schema.IssueList
	Batches []*schema.Batch
	Titles  schema.TitleList

	// titleByLoc holds titles keyed by their location so we don't duplicate the
	// same title entry if it's in the same place.  This is most applicable to
	// live titles, since they're unique per LCCN.
	titleByLoc map[string]*schema.Title

	// Errors represent things wrong with title directories, issue names, batch
	// XML, etc. which are in need of addressing, but which aren't critical
	// enough to halt the rest of the find operation.  These are typically
	// unavoidable human errors we expect to see sometimes, and we need to fix
	// them, but we often still need to know what valid items exist.
	Errors *ErrorList
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
	Errors    *ErrorList

	// This little var helps us answer the age-old question: for a given unique
	// issue, where is it in the workflow?
	IssueNamespace map[*schema.Issue]Namespace
}

// New instantiates a new Finder read to spawn searchers
func New() *Finder {
	return &Finder{
		Searchers: make(map[Namespace]*Searcher),
		Errors: &ErrorList{},
		IssueNamespace: make(map[*schema.Issue]Namespace),
	}
}

// NewSearcher instantiates a Searcher on its own, and typically isn't needed,
// but could be useful for specific one-off scripts
func NewSearcher(ns Namespace, loc string) *Searcher {
	var s = &Searcher{Namespace: ns, Location: loc}
	s.init()
	return s
}

func (s *Searcher) init() {
	s.Issues = make(schema.IssueList, 0)
	s.Batches = make([]*schema.Batch, 0)
	s.Titles = make(schema.TitleList, 0)
	s.titleByLoc = make(map[string]*schema.Title)
	s.Errors = &ErrorList{}
}

func (f *Finder) storeSearcher(s *Searcher) {
	f.Searchers[s.Namespace] = s
	f.aggregate(s)
}

// createAndProcessSearcher instantiates a new Searcher, passes it to
// Processor, aggregates its data in the Finder, and returns the error, if any
func (f *Finder) createAndProcessSearcher(ns Namespace, loc string, processor func(s *Searcher) error) error {
	var s = NewSearcher(ns, loc)
	var err = processor(s)
	f.storeSearcher(s)
	return err
}

// FindDiskBatches creates and runs a custom-namespaced disk batch Searcher,
// aggregates its data, and returns any errors encountered
func (f *Finder) FindDiskBatches(path string) error {
	return f.createAndProcessSearcher(BatchedOnDisk, path, func(s *Searcher) error { return s.FindDiskBatches() })
}

// FindSFTPIssues creates and runs an SFTP Searcher, aggregates its data,
// and returns any errors encountered
func (f *Finder) FindSFTPIssues(path string) error {
	return f.createAndProcessSearcher(SFTPUpload, path, func(s *Searcher) error { return s.FindSFTPIssues() })
}

// FindStandardIssues creates and runs a cutom-namespaced standard issue
// Searcher, aggregates its data, and returns any errors encountered
func (f *Finder) FindStandardIssues(ns Namespace, path string) error {
	return f.createAndProcessSearcher(ns, path, func(s *Searcher) error { return s.FindStandardIssues() })
}

// FindWebBatches creates and runs a website batch Searcher, aggregates its
// data, and returns any errors encountered
func (f *Finder) FindWebBatches(hostname, cachePath string) error {
	return f.createAndProcessSearcher(Website, hostname, func(s *Searcher) error { return s.FindWebBatches(cachePath) })
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
	for _, e := range s.Errors.Errors {
		f.Errors.Append(e)
	}
}
