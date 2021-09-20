package models

import (
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

type auditLogFinder struct {
	cond string
	args []interface{}
	ord  string
	lim  int
}

func (f *auditLogFinder) where(cond string, args ...interface{}) *auditLogFinder {
	f.cond = cond
	f.args = args
	return f
}

func (f *auditLogFinder) limit(limit int) *auditLogFinder {
	f.lim = limit
	return f
}

func (f *auditLogFinder) order(order string) *auditLogFinder {
	f.ord = order
	return f
}

func (f *auditLogFinder) find() ([]*AuditLog, uint64, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var list []*AuditLog
	var selector = op.Select("audit_logs", &AuditLog{})
	if f.cond != "" {
		selector = selector.Where(f.cond, f.args...)
	}
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

// FindRecentAuditLogs returns the most recent limit logs sorted in
// reverse-chronological order. If limit is less than 1, all logs are returned.
// A count is also returned since the most common use-case is displaying a
// subset of records but reporting a total as well.
func FindRecentAuditLogs(limit int) ([]*AuditLog, uint64, error) {
	return new(auditLogFinder).order("`when` desc").limit(limit).find()
}
