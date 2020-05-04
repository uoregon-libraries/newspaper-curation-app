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
