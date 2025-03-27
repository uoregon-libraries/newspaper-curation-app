package models

import (
	"testing"
	"time"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
)

func TestAuditLogFinder(t *testing.T) {
	dbi.DB = &magicsql.DB{}
	var now = time.Now()
	var then = now.Add(-time.Hour)

	type testCase struct {
		fn        func(*AuditLogFinder) *AuditLogFinder
		expectSQL string
	}

	var prefix = "SELECT id,`when`,ip,user,action,message FROM audit_logs"
	var tests = map[string]testCase{
		"Base": {
			fn:        func(f *AuditLogFinder) *AuditLogFinder { return f },
			expectSQL: "WHERE (`action` <> 'autosave') ORDER BY `when` desc",
		},
		"Between": {
			fn:        func(f *AuditLogFinder) *AuditLogFinder { return f.Between(then, now) },
			expectSQL: "WHERE (`action` <> 'autosave') AND (`when` <= ?) AND (`when` >= ?) ORDER BY `when` desc",
		},
		"ForUser": {
			fn:        func(f *AuditLogFinder) *AuditLogFinder { return f.ForUser("bob") },
			expectSQL: "WHERE (`action` <> 'autosave') AND (`user` = ?) ORDER BY `when` desc",
		},
		"ForActions (single)": {
			fn:        func(f *AuditLogFinder) *AuditLogFinder { return f.ForActions(AuditActionClaim) },
			expectSQL: "WHERE (`action` <> 'autosave') AND (`action` IN (?)) ORDER BY `when` desc",
		},
		"ForActions (multiple)": {
			fn:        func(f *AuditLogFinder) *AuditLogFinder { return f.ForActions(AuditActionClaim, AuditActionUnclaim) },
			expectSQL: "WHERE (`action` <> 'autosave') AND (`action` IN (?,?)) ORDER BY `when` desc",
		},
		"Limit": {
			fn:        func(f *AuditLogFinder) *AuditLogFinder { return f.Limit(50) },
			expectSQL: "WHERE (`action` <> 'autosave') ORDER BY `when` desc LIMIT 50",
		},
		"Complex": {
			fn: func(f *AuditLogFinder) *AuditLogFinder {
				return f.Between(then, now).ForUser("jane").ForActions(AuditActionApproveMetadata, AuditActionRejectMetadata).Limit(10)
			},
			expectSQL: "WHERE (`action` <> 'autosave') AND (`action` IN (?,?)) AND (`user` = ?) AND (`when` <= ?) AND (`when` >= ?) ORDER BY `when` desc LIMIT 10",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var f = AuditLogs()
			f = tc.fn(f)
			var got = f.selector().SQL()
			if got != prefix+" "+tc.expectSQL {
				t.Errorf("AuditLogFinder SQL mismatch: got %q, expected %q", got, tc.expectSQL)
			}
		})
	}
}
