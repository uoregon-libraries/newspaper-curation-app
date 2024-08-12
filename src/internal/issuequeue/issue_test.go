package issuequeue

import (
	"math"
	"testing"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

var (
	now              = time.Now()
	goodDate         = "2017-01-01"
	tooRecent        = now.AddDate(0, 0, -10).Format("2006-01-02")
	invalidDate      = "blargh"
	lccnSimple       = "lccn1"
	lccnEmbargoed    = "lccn2"
	lccnNotValidated = "lccn3"
	badlccn          = "badlccn"
	embargoPeriod    = "30 days"
)

var testTitleList = models.TitleList{
	&models.Title{LCCN: lccnSimple, ValidLCCN: true},
	&models.Title{LCCN: lccnEmbargoed, EmbargoPeriod: embargoPeriod, ValidLCCN: true},
	&models.Title{LCCN: lccnNotValidated, ValidLCCN: false},
}

func makeIssue(lccn, date string) *models.Issue {
	var dbi = models.NewIssue("oru", lccn, date, 1)
	dbi.MetadataApprovedAt = now
	return dbi
}

func mustWrap(dbi *models.Issue, t *testing.T) *Issue {
	var i, err = wrapIssue(testTitleList, dbi)
	if err != nil {
		t.Errorf("Error wrapping issue: %s", err)
	}

	return i
}

func TestWrapIssueTableDriven(t *testing.T) {
	type testCase struct {
		description        string
		lccn               string
		date               string
		expectError        bool
		expectEmbargoed    bool
		metadataApprovedAt time.Time
		expectDaysStale    float64
	}

	var twentyDaysAgo = now.AddDate(0, 0, -20)
	var tenYearsAgo = now.AddDate(-10, 0, 0)
	var goodDT, _ = time.Parse("2006-01-02", goodDate)

	var tests = []testCase{
		{description: "Issue with bad lccn",
			lccn: badlccn, date: goodDate, expectError: true},
		{description: "Issue with bad date",
			lccn: lccnSimple, date: invalidDate, expectError: true},
		{description: "Issue with an LCCN that hasn't been validated",
			lccn: lccnNotValidated, date: goodDate, expectError: true},
		{description: "Good issue on simple LCCN",
			lccn: lccnSimple, date: goodDate},
		{description: "Good issue on embargoed LCCN with old date",
			lccn: lccnEmbargoed, date: goodDate},
		{description: "Good issue on embargoed LCCN with recent date",
			lccn: lccnEmbargoed, date: tooRecent, expectEmbargoed: true},
		{description: "Unembargoed issue approved 20 days ago should be stale",
			lccn: lccnSimple, date: tooRecent, metadataApprovedAt: twentyDaysAgo, expectDaysStale: 20},
		{description: "Embargoed issue approved 20 days ago shouldn't be stale",
			lccn: lccnEmbargoed, date: tooRecent, expectEmbargoed: true, metadataApprovedAt: twentyDaysAgo},
		{description: "Embargoed issue with old date and extremely old approval",
			lccn: lccnEmbargoed, date: goodDate, metadataApprovedAt: tenYearsAgo, expectDaysStale: now.Sub(goodDT).Hours()/24 - 30},
		{description: "Unembargoed issue with extremely old approval date",
			lccn: lccnSimple, date: goodDate, metadataApprovedAt: tenYearsAgo, expectDaysStale: now.Sub(tenYearsAgo).Hours() / 24},
	}

	for _, tc := range tests {
		t.Run(tc.description, func(t *testing.T) {
			var dbi = makeIssue(tc.lccn, tc.date)
			if !tc.metadataApprovedAt.IsZero() {
				dbi.MetadataApprovedAt = tc.metadataApprovedAt
			}

			var i, err = wrapIssue(testTitleList, dbi)
			if tc.expectError && err == nil {
				t.Errorf("Expected an error but didn't get one")
			} else if !tc.expectError && err != nil {
				t.Errorf("Didn't expect an error but got one: %v", err)
			}

			// Skip further checks if an error was expected
			if tc.expectError {
				return
			}

			if i.embargoed != tc.expectEmbargoed {
				t.Errorf("Expected embargoed to be %v, got %v", tc.expectEmbargoed, i.embargoed)
			}

			if i.daysStale >= 0 {
				if math.Round(i.daysStale) != math.Round(tc.expectDaysStale) {
					t.Errorf("Expected days stale to be %v, got %v", tc.expectDaysStale, math.Round(i.daysStale))
				}
			}
		})
	}
}
