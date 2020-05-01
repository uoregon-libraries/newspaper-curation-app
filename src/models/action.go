package models

import (
	"time"

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
	ActionTypeComment           ActionType = "user-comment"
	ActionTypeMetadataRejection ActionType = "metadata-rejection"
	ActionTypeMetadataApproval  ActionType = "metadata-approval"
	ActionTypeMetadataEntry     ActionType = "metadata-entry"
)

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
