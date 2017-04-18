// Package issuefinder sets up a process for finding all issues across the
// filesystem and live sites to allow for other tools to get fairly
// comprehensive information: where in the workflow an issue resides,
// which batches contain a certain LCCN, which issues have dupes, etc.
package issuefinder

import (
	"schema"
)

// Finder is *the* component of the issuefinder package, running the filesystem
// and web queries and providing an API to get the results
type Finder struct {
	Issues  schema.IssueList
	Batches []*schema.Batch
	Titles  []*schema.Title

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

// New instantiates a new Finder ready for searching
func New() *Finder {
	return &Finder{titleByLoc: make(map[string]*schema.Title), Errors: &ErrorList{}}
}
