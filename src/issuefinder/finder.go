// Package issuefinder sets up a process for finding all issues across the
// filesystem and live sites to allow for other tools to get fairly
// comprehensive information: where in the workflow an issue resides,
// which batches contain a certain LCCN, which issues have dupes, etc.
package issuefinder

import (
	"db"
	"schema"
)

// Finder is *the* component of the issuefinder package, running the filesystem
// and web queries and providing an API to get the results
type Finder struct {
	Issues  []*schema.Issue
	Batches []*schema.Batch
	Titles  []*schema.Title

	// titleLookup is a necessary evil; in all live situations, the titles refer
	// to the same piece of information.  We can have duped issues and broken
	// batches, but titles are always unique to their LCCN.  On the filesystem,
	// having separate title entities could be handy, but the inconsistency gets
	// difficult to wrangle.
	titleLookup map[string]*schema.Title

	// Errors represent things wrong with title directories, issue names, batch
	// XML, etc. which are in need of addressing, but which aren't critical
	// enough to halt the rest of the find operation.  These are typically
	// unavoidable human errors we expect to see sometimes, and we need to fix
	// them, but we often still need to know what valid items exist.
	Errors []*Error
}

// New instantiates a new Finder ready for searching
func New() *Finder {
	return &Finder{titleLookup: make(map[string]*schema.Title)}
}

// findTitle looks up the title in the lookup, then the database by directory name and LCCN
func (f *Finder) findTitle(titleName string) *schema.Title {
	var title = f.titleLookup[titleName]
	if title != nil {
		return title
	}

	var dbTitle = db.FindTitleByDirectory(titleName)
	if dbTitle == nil {
		dbTitle = db.FindTitleByLCCN(titleName)
	}
	if dbTitle == nil {
		return nil
	}

	title = &schema.Title{LCCN: dbTitle.LCCN}

	// Store it both by sftp dir and lccn so future lookups are easier
	f.titleLookup[dbTitle.SFTPDir] = title
	f.titleLookup[dbTitle.LCCN] = title

	return title
}

// findOrCreateTitle calls findTitle, and on a nil return, creates a new title.
// This is meant for cases where we know the name is as correct as it can be
// and reporting an error isn't helpful.
func (f *Finder) findOrCreateTitle(titleName string) *schema.Title {
	var title = f.findTitle(titleName)
	if title == nil {
		title = &schema.Title{LCCN: titleName}
		f.titleLookup[titleName] = title
	}

	return title
}
