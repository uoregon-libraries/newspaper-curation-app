package workflowhandler

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// CanValidation is a weird little on-off struct to simplify various checks that
// require more than just a simple role validation.  Simple use-cases can just
// return true/false while the more UI-focused pieces can explain why something
// couldn't be done.
type CanValidation struct {
	User    *models.User
	Prefix  string // Message prefix users will see on validation failure
	Context string // Context for logging / alerts
	Error   error  // Specific error
	Status  int    // HTTP status code if somebody tries to do this when they shouldn't
}

// Can returns a Validation to check a user's ability to do certain things.
// It's sort of like a DSL, and thus I hate it, but it's a handy shortcut to
// centralize various checks that need both boolean responses as well as longer
// explanations, depending on context.
func Can(u *models.User) *CanValidation {
	return &CanValidation{User: u, Status: http.StatusOK}
}

// sendResponse sends the handler our responder and issue if there were no errors in
// whatever validations occurred, otherwise it logs errors and reports the
// failure to the user
func (v *CanValidation) sendResponse(h HandlerFunc, resp *responder.Responder, i *Issue) {
	if v.Error == nil {
		h(resp, i)
		return
	}

	logger.Warnf(v.Context + ": " + v.Error.Error())
	resp.Vars.Alert = template.HTML(v.Prefix + ": " + v.Error.Error())
	resp.Writer.WriteHeader(v.Status)
	resp.Render(responder.Empty)
}

// owns sets up error and message, and returns false, if the wrapped user
// doesn't own the given issue.  This centralizes common validations many other
// checks would otherwise duplicate.
func (v *CanValidation) owns(i *Issue) bool {
	if !i.IsOwned() {
		v.Error = errors.New("issue must be claimed first")
		v.Status = http.StatusBadRequest
		return false
	}

	if i.WorkflowOwnerID != v.User.ID {
		v.Error = errors.New("somebody else owns this issue")
		v.Status = http.StatusBadRequest
		return false
	}

	return true
}

// Claim returns true if a user can claim the given issue:
//
// - The issue must not already be owned
// - The user must be allowed to perform actions on issues in the given issue's
//   workflow step
func (v *CanValidation) Claim(i *Issue) bool {
	v.Prefix = "You cannot claim this issue"
	v.Context = fmt.Sprintf("user %q trying to claim issue %d", v.User.Login, i.ID)

	if i.IsOwned() {
		v.Error = errors.New("already owned by somebody")
		v.Status = http.StatusBadRequest
		return false
	}

	switch i.WorkflowStep {
	case schema.WSReadyForMetadataEntry:
		if !v.User.PermittedTo(privilege.EnterIssueMetadata) {
			v.Error = errors.New("insufficient privileges (cannot enter issue metadata)")
			v.Status = http.StatusForbidden
			return false
		}
	case schema.WSAwaitingMetadataReview:
		if !v.User.PermittedTo(privilege.ReviewIssueMetadata) {
			v.Error = errors.New("insufficient privileges (cannot review issue metadata)")
			v.Status = http.StatusForbidden
			return false
		}
	default:
		v.Error = fmt.Errorf("invalid workflow step: %q", i.WorkflowStep)
		v.Status = http.StatusBadRequest
		return false
	}

	return true
}

// Unclaim returns true if a user can "let go" of the given issue: basically
// anything the user is currently the owner of
func (v *CanValidation) Unclaim(i *Issue) bool {
	v.Prefix = "You cannot unclaim this issue"
	v.Context = fmt.Sprintf("user %q trying to unclaim issue %d", v.User.Login, i.ID)
	return v.owns(i)
}

// EnterMetadata returns true if the user can enter metadata for the given issue:
//
// - The user's role must allow issue metadata entry
// - It must be claimed by this user
// - The issue must be awaiting metadata entry
func (v *CanValidation) EnterMetadata(i *Issue) bool {
	v.Prefix = "You cannot modify this issue's metadata"
	v.Context = fmt.Sprintf("user %q trying to enter metadata for issue %d", v.User.Login, i.ID)

	if !v.User.PermittedTo(privilege.EnterIssueMetadata) {
		v.Error = errors.New("insufficient privileges")
		v.Status = http.StatusForbidden
		return false
	}

	if !v.owns(i) {
		return false
	}

	if i.WorkflowStep != schema.WSReadyForMetadataEntry {
		v.Error = errors.New("issue not awaiting metadata entry")
		v.Status = http.StatusBadRequest
		return false
	}

	return true
}

// ReviewMetadata returns true if the user can review metadata for the given issue:
//
// - The user's role must allow issue metadata review
// - It must be claimed by this user
// - The issue must be awaiting metadata review
// - If the data entry was done by this user, they must be allowed to do self-review
func (v *CanValidation) ReviewMetadata(i *Issue) bool {
	v.Prefix = "You cannot review this issue's metadata"
	v.Context = fmt.Sprintf("user %q trying to review metadata for issue %d", v.User.Login, i.ID)

	if !v.User.PermittedTo(privilege.ReviewIssueMetadata) {
		v.Error = errors.New("insufficient privileges")
		v.Status = http.StatusForbidden
		return false
	}

	if !v.owns(i) {
		return false
	}

	if i.WorkflowStep != schema.WSAwaitingMetadataReview {
		v.Error = fmt.Errorf("issue not awaiting metadata review (workflow step: %s)", i.WorkflowStep)
		v.Status = http.StatusBadRequest
		return false
	}

	if i.MetadataEntryUserID == v.User.ID && !v.User.PermittedTo(privilege.ReviewOwnMetadata) {
		v.Error = fmt.Errorf("author cannot also be reviewer")
		v.Status = http.StatusBadRequest
		return false
	}

	return true
}
