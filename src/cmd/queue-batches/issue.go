package main

import (
	"fmt"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
)

// issue wraps a db Issue but gives us a page count as well as how old this
// issue is *relative to embargoes*
type issue struct {
	*db.Issue
	title     *db.Title
	pages     int
	daysStale float64
	embargoed bool
}

func wrapIssue(dbIssue *db.Issue) (*issue, error) {
	var issueDate, err = time.Parse("2006-01-02", dbIssue.Date)
	if err != nil {
		return nil, fmt.Errorf("%q is an invalid date: %s", dbIssue.Date, err)
	}

	var i = &issue{Issue: dbIssue, pages: len(dbIssue.PageLabels)}

	i.title = titles.Find(i.LCCN)
	if i.title == nil {
		return nil, fmt.Errorf("LCCN %q has no database title", i.LCCN)
	}

	var embargoLiftDate time.Time
	embargoLiftDate, err = i.title.CalculateEmbargoLiftDate(issueDate)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse title's embargo duration: %s", err)
	}

	if embargoLiftDate.After(time.Now()) {
		i.embargoed = true
	}

	// Embargoed issues can be waiting for a while before their embargo is
	// lifted, so we have to consider them stale based on the newer date:
	// metadata approval or embargo lifting.
	if i.MetadataApprovedAt.Before(embargoLiftDate) {
		i.daysStale = time.Since(embargoLiftDate).Hours() / 24.0
	} else {
		i.daysStale = time.Since(i.MetadataApprovedAt).Hours() / 24.0
	}

	return i, nil
}
