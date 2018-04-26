package main

import (
	"db"
	"testing"
	"time"
)

var (
	now           = time.Now()
	goodDate      = "2017-01-01"
	tooRecent     = now.AddDate(0, 0, -10).Format("2006-01-02")
	invalidDate   = "blargh"
	lccnSimple    = "lccn1"
	lccnEmbargoed = "lccn2"
	badlccn       = "badlccn"
	embargoedDays = 30
)

func overrideLookup() {
	lookupTitle = func(s string) *db.Title {
		switch s {
		case lccnSimple:
			return &db.Title{}
		case lccnEmbargoed:
			return &db.Title{Embargoed: true}
		default:
			return nil
		}
	}
}

func makeIssue(lccn, date string) *db.Issue {
	var dbi = db.NewIssue("oru", lccn, date, 1)
	dbi.MetadataApprovedAt = now
	return dbi
}

func mustWrap(dbi *db.Issue, t *testing.T) *issue {
	var i, err = wrapIssue(dbi, embargoedDays)
	if err != nil {
		t.Errorf("Error wrapping issue: %s", err)
	}

	return i
}

func TestWrapIssue(t *testing.T) {
	overrideLookup()

	var dbi *db.Issue
	var i *issue
	var err error

	dbi = makeIssue(badlccn, goodDate)
	i, err = wrapIssue(dbi, embargoedDays)
	if err == nil {
		t.Errorf("Issue with bad lccn shouldn't have worked")
	}
	t.Logf("Got error (this is expected): %s", err)

	dbi = makeIssue(lccnSimple, invalidDate)
	i, err = wrapIssue(dbi, embargoedDays)
	if err == nil {
		t.Errorf("Issue with bad date shouldn't have worked")
	}
	t.Logf("Got error (this is expected): %s", err)

	dbi = makeIssue(lccnSimple, goodDate)
	i = mustWrap(dbi, t)
	if i.embargoed {
		t.Errorf("Good issue on simple LCCN is somehow embargoed")
	}

	dbi = makeIssue(lccnEmbargoed, goodDate)
	i = mustWrap(dbi, t)
	if i.embargoed {
		t.Errorf("Good issue on embargoed LCCN (with an old date) is somehow embargoed")
	}

	dbi = makeIssue(lccnEmbargoed, tooRecent)
	i = mustWrap(dbi, t)
	if !i.embargoed {
		t.Errorf("Good issue on embargoed LCCN (with a recent date) is not embargoed")
	}

	dbi = makeIssue(lccnSimple, tooRecent)
	var twentyDaysAgo = time.Now().AddDate(0, 0, -20)
	dbi.MetadataApprovedAt = twentyDaysAgo
	i = mustWrap(dbi, t)
	if i.daysStale != 20 {
		t.Errorf("Unembargoed issue's days stale is %d; should have been twenty", i.daysStale)
	}

	dbi = makeIssue(lccnEmbargoed, tooRecent)
	dbi.MetadataApprovedAt = twentyDaysAgo
	i = mustWrap(dbi, t)
	if i.daysStale > 0 {
		t.Errorf("Embargoed issue's days stale is %d; should have been negative due to embargo", i.daysStale)
	}

	dbi = makeIssue(lccnEmbargoed, goodDate)
	i = mustWrap(dbi, t)
	if i.daysStale != 0 {
		t.Errorf("Embargoed issue (with old date) should have been stale for 0 days")
	}

	var gdt, _ = time.Parse("2006-01-02", goodDate)
	var expectedStale = int(now.Sub(gdt).Hours()/24) - embargoedDays
	dbi = makeIssue(lccnEmbargoed, goodDate)
	dbi.MetadataApprovedAt = now.AddDate(-10, 0, 0)
	i = mustWrap(dbi, t)
	t.Logf("Expecting %d stale days", expectedStale)
	if i.daysStale != expectedStale {
		t.Errorf("Embargoed issue (with old date and extremely old approval date) was stale for %d days, "+
			"but should have been stale for %d days", i.daysStale, expectedStale)
	}

	dbi = makeIssue(lccnSimple, goodDate)
	dbi.MetadataApprovedAt = now.AddDate(-10, 0, 0)
	expectedStale = int(now.Sub(dbi.MetadataApprovedAt).Hours() / 24)
	i = mustWrap(dbi, t)
	t.Logf("Expecting %d stale days", expectedStale)
	if i.daysStale != expectedStale {
		t.Errorf("Unembargoed issue (with extremely old approval date) was stale for %d days, "+
			"but should have been stale for %d days", i.daysStale, expectedStale)
	}
}
