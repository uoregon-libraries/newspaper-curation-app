// cached_finder.go creates types which are used when converting a Finder into
// something we can actually cache to disk

package issuefinder

import (
	"apperr"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

// cacheID is just a uint used to make it clear there's a relationship being expressed
type cacheID uint

// cachedFinder, and cached* "children", are serialization-friendly structures
// which allows us to store and retrieve issue, title, and batch data without
// the recursive problems which exist due to things like batches having a list
// of issues while issues store their batch
type cachedFinder struct {
	Searchers []cachedSearcher
}

type cachedSearcher struct {
	Namespace Namespace
	Location  string
	Batches   []cachedBatch
	Titles    []cachedTitle
	Issues    []cachedIssue
	Errors    apperr.List
}

type cachedBatch struct {
	ID          cacheID
	MARCOrgCode string
	Keyword     string
	Version     int
	Location    string
	Errors      apperr.List
}

type cachedTitle struct {
	ID                 cacheID
	LCCN               string
	Name               string
	PlaceOfPublication string
	Location           string
	Errors             apperr.List
}

type cachedIssue struct {
	ID           cacheID
	TitleID      cacheID
	RawDate      string
	Edition      int
	BatchID      cacheID
	Location     string
	WorkflowStep string
	Files        []cachedFile
	Errors       apperr.List
}

type cachedFile struct {
	fileutil.File
	ID       cacheID
	Location string
	Errors   apperr.List
}
