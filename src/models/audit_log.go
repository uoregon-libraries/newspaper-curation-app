package models

import (
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
)

// AuditLog represents the audit_logs table
type AuditLog struct {
	ID      int       `sql:",primary"`
	When    time.Time "sql:\"`when`\""
	IP      string
	User    string
	Action  string
	Message string
}

// CreateAuditLog writes the given data to audit_logs
func CreateAuditLog(ip, user, action, message string) error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Save("audit_logs", &AuditLog{When: time.Now(), IP: ip, User: user, Action: action, Message: message})
	return op.Err()
}

// AuditLogFinder is a pseudo-DSL for easily creating queries without needing
// to know the underlying table structure
type AuditLogFinder struct {
	// this looks weird, but making a map of conditions allows us to have helpers
	// that just replace data instead of having to worry about deduping it. e.g.,
	// if somebody calls f.ForUser("foo").ForUser("bar")
	conditions map[string]interface{}
	ord        string
	lim        int
}

func (f *AuditLogFinder) order(order string) *AuditLogFinder {
	f.ord = order
	return f
}

// AuditLogs returns a scoped object for use in simple filtering of the
// audit_logs table without needing manual SQL or deep knowledge of the
// database. It is meant to be ORM-like but with a very narrow scope:
//
//   AuditLogs().Between(time.Date(), time.Now()).ForUser("jechols").Limit(100).All()
func AuditLogs() *AuditLogFinder {
	var f = &AuditLogFinder{conditions: make(map[string]interface{})}
	f.conditions["action <> 'autosave'"] = nil
	f.ord = "`when` desc"
	return f
}

// Between returns a scoped finder for limiting the results of the query to >=
// start and <= end.
func (f *AuditLogFinder) Between(start, end time.Time) *AuditLogFinder {
	f.conditions["`when` >= ?"] = start
	f.conditions["`when` <= ?"] = end
	return f
}

// ForUser scopes the finder to only look for logs where the given string is in
// the username field
func (f *AuditLogFinder) ForUser(u string) *AuditLogFinder {
	f.conditions["`user` = ?"] = u
	return f
}

// Limit makes f.All() return at most limit AuditLog instances
func (f *AuditLogFinder) Limit(limit int) *AuditLogFinder {
	f.lim = limit
	return f
}

// All returns all logs for the current query. If a limit was set, the returned
// AuditLog objects will be limited, but the second return value will indicate
// how many total logs there were.
func (f *AuditLogFinder) All() ([]*AuditLog, uint64, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var list []*AuditLog

	var where []string
	var args []interface{}
	for k, v := range f.conditions {
		where = append(where, k)
		if v != nil {
			args = append(args, v)
		}
	}
	var selector = op.Select("audit_logs", &AuditLog{}).Where(strings.Join(where, " AND "), args...)

	if f.ord != "" {
		selector = selector.Order(f.ord)
	}
	var num = selector.Count().RowCount()
	if f.lim > 0 {
		selector = selector.Limit(uint64(f.lim))
	}
	selector.AllObjects(&list)

	return list, num, op.Err()
}
