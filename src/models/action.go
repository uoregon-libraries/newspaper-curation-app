package models

import (
	"time"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
)

// Object types for consistency in the database
const (
	actionObjectTypeIssue = "issue"
)

// ActionType holds machine-friendly text telling us what kind of action we
// have
type ActionType string

// Full list of valid action types
const (
	ActionTypeComment              ActionType = "user-comment"
	ActionTypeInternalProcess      ActionType = "internal-process"
	ActionTypeMetadataRejection    ActionType = "metadata-rejection"
	ActionTypeMetadataApproval     ActionType = "metadata-approval"
	ActionTypeMetadataEntry        ActionType = "metadata-entry"
	ActionTypeReportUnfixableError ActionType = "report-unfixable-error"
	ActionTypeReturnCurate         ActionType = "return-metadata-entry"
	ActionTypeReturnReview         ActionType = "return-metadata-review"
	ActionTypeRemoveErrorIssue     ActionType = "remove-error-issue"
	ActionTypeClaim                ActionType = "claim-issue"
	ActionTypeUnclaim              ActionType = "unclaim-issue"
)

// Describe gives a human-readable explanation of what happened when a given
// action type was applied
func (at ActionType) Describe() string {
	switch at {
	case ActionTypeComment:
		return "wrote a comment"
	case ActionTypeInternalProcess:
		return "executed an internal process"
	case ActionTypeMetadataRejection:
		return "rejected the issue's metadata"
	case ActionTypeMetadataApproval:
		return "approved the issue's metadata"
	case ActionTypeMetadataEntry:
		return "added metadata and pushed the issue to review"
	case ActionTypeReportUnfixableError:
		return "reported an unfixable error"
	case ActionTypeReturnCurate:
		return "returned the issue for metadata entry"
	case ActionTypeReturnReview:
		return "returned the issue for metadata review"
	case ActionTypeRemoveErrorIssue:
		return "moved the issue from NCA to the error folder"
	case ActionTypeClaim:
		return "claimed the issue"
	case ActionTypeUnclaim:
		return "removed the issue from the prior owner's desk"
	default:
		return string(at)
	}
}

// Action holds onto information about an object (issues for now) so
// communication can be centralized in NCA and be easily visible when, for
// instance, curators need to respond to rejection notes.
type Action struct {
	ID         int       `sql:",primary"`
	CreatedAt  time.Time // When was it created
	ObjectType string    // "issue" for instance
	ObjectID   int       // Issue id / batch id / etc
	ActionType string    // Issue metadata rejection, User comment, etc.
	UserID     int       // Who created the action
	Message    string    // Free-text message

	user *User
}

func newAction() *Action {
	return &Action{CreatedAt: time.Now()}
}

// newIssueAction returns an action pre-filled with some basic issue metadata
func newIssueAction(id int, aType ActionType) *Action {
	var a = newAction()
	a.ObjectType = actionObjectTypeIssue
	a.ActionType = string(aType)
	a.ObjectID = id

	return a
}

func findActionsByObjectTypeAndID(oType string, oID int) ([]*Action, error) {
	var list []*Action
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Select("actions", &Action{}).
		Where("object_type = ? AND object_id = ?", oType, oID).
		Order("created_at asc").
		AllObjects(&list)

	return list, op.Err()
}

// FindActionsForIssue returns all actions for the given issue id sorted
// oldest first
func FindActionsForIssue(issueID int) ([]*Action, error) {
	return findActionsByObjectTypeAndID(actionObjectTypeIssue, issueID)
}

// Author returns the action author
func (a *Action) Author() *User {
	if a.user == nil {
		a.user = FindUserByID(a.UserID)
	}
	return a.user
}

// Type returns a converted ActionType rather than the stringified version
func (a *Action) Type() ActionType {
	return ActionType(a.ActionType)
}

// Save creates or updates the Action in the actions table
func (a *Action) Save() error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	return a.SaveOp(op)
}

// important returns true for actions that really need to be seen, based on
// their type.  This allows us to exclude things we want to capture, but not
// spam at the curators / reviewers.
func (a *Action) important() bool {
	switch ActionType(a.ActionType) {
	case ActionTypeInternalProcess, ActionTypeClaim, ActionTypeUnclaim:
		return false
	}

	return true
}

// SaveOp creates or updates the Action with a custom operation (e.g., for
// transaction-dependent saves)
func (a *Action) SaveOp(op *magicsql.Operation) error {
	op.Save("actions", a)
	return op.Err()
}
