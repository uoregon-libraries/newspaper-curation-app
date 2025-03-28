package models

import (
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// FlatIssue is our data model which combines "core" issue metadata with batch
// and title data into a single entity. This relies on a pretty specific view
// (`flat_issues`) to grab a bunch of live issues so this data model can be
// simpler to work with than more generic models being joined together.
//
// Because of the current use-case, this model only holds data that's useful
// for *live* issues. The view has more data, but we aren't bothering to pull
// it until we find a need to do so.
//
// This is a data-only model, and part of an ongoing effort to separate data
// structures from business-logic structures.
type FlatIssue struct {
	// Issue fields
	ID            int64 `sql:",primary"`
	MARCOrgCode   string
	LCCN          string
	Date          string
	DateAsLabeled string
	Volume        string
	Issue         string
	Edition       int
	EditionLabel  string
	HumanName     string
	BatchID       int64
	PageCount     int

	// Title fields
	TitleName string
	MARCTitle string

	// Batch fields, though the live/archive data is obviously also when the
	// issue went live or was archived
	BatchName     string
	BatchFullName string
	WentLiveAt    time.Time
	ArchivedAt    time.Time
}

// FlatIssueFinder is our mini-DSL for the `flat_issues` view. It mirrors many
// of IssueFinder's filter options, but for now is geared for finding live
// issues that may need to be pulled.
type FlatIssueFinder struct {
	*coreFinder[*FlatIssueFinder]
}

// FlatIssues returns a FlatIssueFinder
func FlatIssues() *FlatIssueFinder {
	var f = &FlatIssueFinder{}
	f.coreFinder = newCoreFinder(f, "flat_issues", &FlatIssue{})
	return f
}

// Live returns a scope for issues in production
func (f *FlatIssueFinder) Live() *FlatIssueFinder {
	f.conditions["workflow_step = ?"] = schema.WSInProduction
	return f
}

// LCCN returns a scope for finding issues with a particular title
func (f *FlatIssueFinder) LCCN(lccn string) *FlatIssueFinder {
	f.conditions["lccn = ?"] = lccn
	return f
}

// MOC returns a scope for finding issues with a particular awardee (MARC Org Code)
func (f *FlatIssueFinder) MOC(moc string) *FlatIssueFinder {
	f.conditions["marc_org_code = ?"] = moc
	return f
}

// Date looks for an issue that was published on the given date
func (f *FlatIssueFinder) Date(date string) *FlatIssueFinder {
	f.conditions["date = ?"] = date
	return f
}

// WentLiveBetween returns a scoped finder for limiting the results of the
// query to issues which went live on or after start, and on or before end
func (f *FlatIssueFinder) WentLiveBetween(start, end time.Time) *FlatIssueFinder {
	f.conditions["went_live_at >= ?"] = start
	f.conditions["went_live_at <= ?"] = end
	return f
}

// Fetch returns all issues this scoped finder represents
func (f *FlatIssueFinder) Fetch() ([]*FlatIssue, error) {
	var list []*FlatIssue
	var err = f.coreFinder.Fetch(&list)
	return list, err
}
