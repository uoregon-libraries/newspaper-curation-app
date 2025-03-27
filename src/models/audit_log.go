package models

import (
	"fmt"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
)

// AuditAction is a semi-controlled integer representing the possible audit log
// action types. This hack is what bad devs like me end up doing when they
// don't properly normalize their data from the beginning.
type AuditAction uint8

// All currently valid audit log actions
const (
	AuditActionUnderflow AuditAction = iota

	AuditActionQueue
	AuditActionSaveTitle
	AuditActionValidateTitle
	AuditActionCreateMoc
	AuditActionUpdateMoc
	AuditActionDeleteMoc
	AuditActionSaveUser
	AuditActionDeactivateUser
	AuditActionClaim
	AuditActionUnclaim
	AuditActionApproveMetadata
	AuditActionRejectMetadata
	AuditActionReportError
	AuditActionUndoErrorIssue
	AuditActionRemoveErrorIssue
	AuditActionQueueForReview
	AuditActionAutosave
	AuditActionSaveDraft
	AuditActionSaveQueue
	AuditActionUploadMARC

	AuditActionOverflow
)

var dbAuditActions = map[AuditAction]string{
	AuditActionQueue:            "queue",
	AuditActionSaveTitle:        "save-title",
	AuditActionValidateTitle:    "validate-title",
	AuditActionCreateMoc:        "create-moc",
	AuditActionUpdateMoc:        "update-moc",
	AuditActionDeleteMoc:        "delete-moc",
	AuditActionSaveUser:         "save-user",
	AuditActionDeactivateUser:   "deactivate-user",
	AuditActionClaim:            "claim",
	AuditActionUnclaim:          "unclaim",
	AuditActionApproveMetadata:  "approve-metadata",
	AuditActionRejectMetadata:   "reject-metadata",
	AuditActionReportError:      "report-error",
	AuditActionUndoErrorIssue:   "undo-error-issue",
	AuditActionRemoveErrorIssue: "remove-error-issue",
	AuditActionQueueForReview:   "queue-for-review",
	AuditActionAutosave:         "autosave",
	AuditActionSaveDraft:        "savedraft",
	AuditActionSaveQueue:        "savequeue",
	AuditActionUploadMARC:       "upload-marc",
}

// String returns the human-readable value for an action
func (a AuditAction) String() string {
	return dbAuditActions[a]
}

var auditActionLookup = map[string]AuditAction{
	"queue":              AuditActionQueue,
	"save-title":         AuditActionSaveTitle,
	"validate-title":     AuditActionValidateTitle,
	"create-moc":         AuditActionCreateMoc,
	"update-moc":         AuditActionUpdateMoc,
	"delete-moc":         AuditActionDeleteMoc,
	"save-user":          AuditActionSaveUser,
	"deactivate-user":    AuditActionDeactivateUser,
	"claim":              AuditActionClaim,
	"unclaim":            AuditActionUnclaim,
	"approve-metadata":   AuditActionApproveMetadata,
	"reject-metadata":    AuditActionRejectMetadata,
	"report-error":       AuditActionReportError,
	"undo-error-issue":   AuditActionUndoErrorIssue,
	"remove-error-issue": AuditActionRemoveErrorIssue,
	"queue-for-review":   AuditActionQueueForReview,
	"autosave":           AuditActionAutosave,
	"savedraft":          AuditActionSaveDraft,
	"savequeue":          AuditActionSaveQueue,
	"upload-marc":        AuditActionUploadMARC,
}

// AuditActionFromString returns the action int for the given string, if the
// string is one of our known actions
func AuditActionFromString(s string) AuditAction {
	return auditActionLookup[s]
}

// AuditLog represents the audit_logs table
type AuditLog struct {
	ID      int64     `sql:",primary"`
	When    time.Time "sql:\"`when`\""
	IP      string
	User    string
	Action  string
	Message string
}

// CreateAuditLog writes the given data to audit_logs
func CreateAuditLog(ip, user string, action AuditAction, message string) error {
	var alog, err = BuildAuditLog(ip, user, action, message)
	if err != nil {
		return err
	}
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Save("audit_logs", alog)
	return op.Err()
}

// BuildAuditLog creates the data for an audit log without saving it
func BuildAuditLog(ip, user string, action AuditAction, message string) (*AuditLog, error) {
	if action <= AuditActionUnderflow || action >= AuditActionOverflow {
		return nil, fmt.Errorf("Unknown audit action")
	}
	return &AuditLog{When: time.Now(), IP: ip, User: user, Action: action.String(), Message: message}, nil
}

// AuditLogFinder is a pseudo-DSL for easily creating queries without needing
// to know the underlying table structure
type AuditLogFinder struct {
	*coreFinder
}

// AuditLogs returns a scoped object for use in simple filtering of the
// audit_logs table without needing manual SQL or deep knowledge of the
// database. It is meant to be ORM-like but with a very narrow scope:
//
//	AuditLogs().Between(time.Date(), time.Now()).ForUser("jechols").Limit(100).Fetch()
func AuditLogs() *AuditLogFinder {
	var f = newCoreFinder("audit_logs", &AuditLog{})
	f.conditions["action <> 'autosave'"] = nil
	f.ord = "`when` desc"
	return &AuditLogFinder{coreFinder: f}
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

// ForActions scopes the finder to a specific list of actions.
func (f *AuditLogFinder) ForActions(list ...AuditAction) *AuditLogFinder {
	var dbActions = make([]any, len(list))
	for i, action := range list {
		dbActions[i] = dbAuditActions[action]
	}

	// Use the magic "(??)" syntax so coreFinder handles the slice properly
	f.conditions["`action` IN (??)"] = dbActions
	return f
}

// Limit makes f.Fetch() return at most limit AuditLog instances
func (f *AuditLogFinder) Limit(limit int) *AuditLogFinder {
	f.lim = limit
	return f
}

// Fetch returns all logs for the current query. If a limit was set, the returned
// AuditLog objects will be limited, but the second return value will indicate
// how many total logs there were.
func (f *AuditLogFinder) Fetch() ([]*AuditLog, uint64, error) {
	var num, err = f.coreFinder.Count()
	if err != nil {
		return nil, 0, err
	}

	var list []*AuditLog
	err = f.coreFinder.Fetch(&list)
	if err != nil {
		return nil, 0, err
	}

	return list, num, nil
}
