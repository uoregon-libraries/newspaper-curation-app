package models

import (
	"strings"
	"testing"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

func TestIssueFinder(t *testing.T) {
	dbi.DB = &magicsql.DB{}

	type testCase struct {
		fn        func(*IssueFinder) *IssueFinder
		expectSQL string
	}

	var prefix = "SELECT id,marc_org_code,lccn,date,date_as_labeled,volume,issue,edition,edition_label,page_labels_csv,page_count,batch_id,location,backup_location,human_name,is_from_scanner,workflow_step,workflow_owner_id,workflow_owner_expires_at,metadata_entry_user_id,metadata_entered_at,reviewed_by_user_id,metadata_approved_at,rejected_by_user_id,ignored,draft_comment FROM issues"
	var tests = map[string]testCase{
		"Base": {
			fn:        func(f *IssueFinder) *IssueFinder { return f },
			expectSQL: "%PREFIX% WHERE (ignored = ?)",
		},
		"LCCN": {
			fn:        func(f *IssueFinder) *IssueFinder { return f.LCCN("sn12345678") },
			expectSQL: "%PREFIX% WHERE (ignored = ?) AND (lccn = ?)",
		},
		"MOC": {
			fn:        func(f *IssueFinder) *IssueFinder { return f.MOC("oru") },
			expectSQL: "%PREFIX% WHERE (ignored = ?) AND (marc_org_code = ?)",
		},
		"InWorkflowStep": {
			fn:        func(f *IssueFinder) *IssueFinder { return f.InWorkflowStep(schema.WSReadyForBatching) },
			expectSQL: "%PREFIX% WHERE (ignored = ?) AND (workflow_step = ?)",
		},
		"BatchID": {
			fn:        func(f *IssueFinder) *IssueFinder { return f.BatchID(101) },
			expectSQL: "%PREFIX% WHERE (batch_id = ?) AND (ignored = ?)",
		},
		"AllowIgnored": {
			fn:        func(f *IssueFinder) *IssueFinder { return f.AllowIgnored() },
			expectSQL: "%PREFIX%",
		},
		"OnDesk": {
			fn:        func(f *IssueFinder) *IssueFinder { return f.OnDesk(5) },
			expectSQL: "%PREFIX% WHERE (ignored = ?) AND (workflow_owner_expires_at > ?) AND (workflow_owner_expires_at IS NOT NULL) AND (workflow_owner_id = ?)",
		},
		"Available": {
			fn:        func(f *IssueFinder) *IssueFinder { return f.Available() },
			expectSQL: "%PREFIX% WHERE (ignored = ?) AND (workflow_owner_id = 0 OR workflow_owner_expires_at < ?)",
		},
		"NotCuratedBy": {
			fn:        func(f *IssueFinder) *IssueFinder { return f.NotCuratedBy(10) },
			expectSQL: "%PREFIX% WHERE (ignored = ?) AND (metadata_entry_user_id <> ?)",
		},
		"Limit": {
			fn:        func(f *IssueFinder) *IssueFinder { return f.Limit(25) },
			expectSQL: "%PREFIX% WHERE (ignored = ?) LIMIT 25",
		},
		"OrderBy": {
			fn:        func(f *IssueFinder) *IssueFinder { return f.OrderBy("lccn asc, date desc") },
			expectSQL: "%PREFIX% WHERE (ignored = ?) ORDER BY lccn asc, date desc",
		},
		"Complex": {
			fn: func(f *IssueFinder) *IssueFinder {
				return f.LCCN("sn123").MOC("oru").InWorkflowStep(schema.WSAwaitingMetadataReview).Available().NotCuratedBy(7).OrderBy("date").Limit(100)
			},
			expectSQL: "%PREFIX% WHERE (ignored = ?) AND (lccn = ?) AND (marc_org_code = ?) AND (metadata_entry_user_id <> ?) AND (workflow_owner_id = 0 OR workflow_owner_expires_at < ?) AND (workflow_step = ?) ORDER BY date LIMIT 100",
		},
		"date/edition": {
			fn: func(f *IssueFinder) *IssueFinder {
				return f.date("2024-03-27").edition(1)
			},
			expectSQL: "%PREFIX% WHERE (date = ?) AND (edition = ?) AND (ignored = ?)",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var f = Issues()
			f = tc.fn(f)
			var got = f.selector().SQL()
			if got != strings.Replace(tc.expectSQL, "%PREFIX%", prefix, 1) {
				t.Errorf("IssueFinder SQL mismatch: got %q, expected %q", got, tc.expectSQL)
			}
		})
	}
}
