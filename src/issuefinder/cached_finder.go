// cached_finder.go creates types which are used when converting a Finder into
// something we can actually cache to disk

package issuefinder

import (
	"fileutil"
	"time"
)

// cacheID is just a uint used to make it clear there's a relationship being expressed
type cacheID uint

// cachedFinder, and cached* "children", are serialization-friendly structures
// which allows us to store and retrieve issue, title, and batch data without
// the recursive problems which exist due to things like batches having a list
// of issues while issues store their batch
type cachedFinder struct {
	Batches []cachedBatch
	Titles  []cachedTitle
	Issues  []cachedIssue
	Errors  []cachedError
}

type cachedBatch struct {
	ID          cacheID
	MARCOrgCode string
	Keyword     string
	Version     int
	Location    string
}

type cachedTitle struct {
	ID                 cacheID
	LCCN               string
	Name               string
	PlaceOfPublication string
	Location           string
}

type cachedIssue struct {
	ID       cacheID
	TitleID  cacheID
	Date     time.Time
	Edition  int
	BatchID  cacheID
	Location string
	Files    []cachedFile
}

type cachedFile struct {
	fileutil.File
	ID       cacheID
	Location string
}

type cachedError struct {
	BatchID  cacheID
	TitleID  cacheID
	IssueID  cacheID
	FileID   cacheID
	Location string
	Error    string
}
