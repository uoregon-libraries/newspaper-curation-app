package main

import (
	"db"
	"fmt"
	"time"
)

// issue wraps a db Issue but gives us a page count as well as how old this
// issue is *relative to embargoes*
type issue struct {
	*db.Issue
	title     *db.Title
	pages     int
	daysStale int
	embargoed bool
}

func wrapIssue(dbIssue *db.Issue, embargoedDays int) (*issue, error) {
	var issueDate, err = time.Parse("2006-01-02", dbIssue.Date)
	if err != nil {
		return nil, fmt.Errorf("%q is an invalid date: %s", dbIssue.Date, err)
	}

	var i = &issue{Issue: dbIssue, pages: len(dbIssue.PageLabels)}

	i.title = titles.Find(i.LCCN)
	if i.title == nil {
		return nil, fmt.Errorf("LCCN %q has no database title", i.LCCN)
	}

	// How many days has this issue been waiting to be batched?
	i.daysStale = int(time.Since(i.MetadataApprovedAt).Hours() / 24.0)

	if i.title.Embargoed {
		var embargoLiftDate = issueDate.Add(time.Hour * time.Duration(24*embargoedDays))
		if embargoLiftDate.After(time.Now()) {
			i.embargoed = true
		}

		// Embargoed issues can be waiting for a while before their embargo is
		// lifted, so we have to consider them stale based on the newer date:
		// metadata approval or embargo lifting.
		if i.MetadataApprovedAt.Before(embargoLiftDate) {
			i.daysStale = int(time.Since(embargoLiftDate).Hours() / 24.0)
		}
	}

	return i, nil
}
