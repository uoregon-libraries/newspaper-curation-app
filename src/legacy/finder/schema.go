// schema.go: simple data types for our title and issue finding code to use,
// isolated here so we can more easily reuse this if it makes sense later.

package main

import (
	"fmt"
	"time"
)

// Title represents whatever common data we need across titles we read from
// filesystem data, database data, and the live site.  Note that each Title
// instance is a single *source* of Title data, so we could have multiple
// instances of the same LCCN simply by scanning in multiple locations.  In
// typical cases, each one will have a unique list of issues, but duped issues
// can also exist when something goes wrong.
type Title struct {
	// LCCN should be set for all titles regardless where they're found.  If we
	// read data from an SFTP directory, this will have to be looked up in the
	// database, as those are named for the SFTP user and won't have an LCCN just
	// from the filesystem data.
	LCCN string
	Issues []*Issue
}

// AppendIssue creates an issue under this title, sets up its date and edition
// number, and returns it
func (t *Title) AppendIssue(date time.Time, ed int) *Issue {
	var i = &Issue{Date: date, Edition: ed, Title: t}
	t.Issues = append(t.Issues, i)
	return i
}

// Issue is an extremely basic encapsulation of an issue's high-level data
type Issue struct {
	Date    time.Time
	Edition int
	Title   *Title
}

// Key returns the unique string that represents this issue
func (i *Issue) Key() string {
	return fmt.Sprintf("%s/%s%02d", i.Title.LCCN, i.Date.Format("20050102"), i.Edition)
}
