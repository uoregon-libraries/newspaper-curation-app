package workflowhandler

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db/user"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// Handler is our version of http.Handler for sending extra context to
// workflow handlers
type Handler interface {
	ServeHTTP(*responder.Responder, *Issue)
}

// HandlerFunc represents workflow handlers with workflow-specific context
type HandlerFunc func(resp *responder.Responder, i *Issue)

// ServeHTTP calls f(resp, i)
func (f HandlerFunc) ServeHTTP(resp *responder.Responder, i *Issue) {
	f(resp, i)
}

// handle wraps the http package's middleware magic to let us send the
// responder and issue (if any) to the Handlers so we know all the
// database hits are out of the way
func handle(h HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resp = responder.Response(w, r)
		var u = resp.Vars.User
		var idStr = mux.Vars(r)["issue_id"]
		var issue *Issue

		// If there's an issue_id parameter, we validate it - unless somebody
		// deliberately hacks a URL, none of these errors should be very likely
		if idStr != "" {
			var id, _ = strconv.Atoi(idStr)
			if id == 0 {
				logger.Warnf("Invalid issue id requested by %s: %s", u.Login, idStr)
				resp.Vars.Alert = "Invalid issue"
				w.WriteHeader(http.StatusBadRequest)
				resp.Render(responder.Empty)
				return
			}

			var i, err = db.FindIssue(id)
			if err != nil {
				logger.Errorf("Error trying to look up issue id %d: %s", id, err)
				resp.Vars.Alert = "Database error; try again or contact the system administrator"
				w.WriteHeader(http.StatusInternalServerError)
				resp.Render(responder.Empty)
				return
			}

			if i == nil {
				logger.Warnf("User %s trying to find nonexistent issue id %d", u.Login, id)
				resp.Vars.Alert = "Issue not found; try again or contact the system administrator"
				w.WriteHeader(http.StatusNotFound)
				resp.Render(responder.Empty)
				return
			}

			issue = wrapDBIssue(i)
		}

		h.ServeHTTP(resp, issue)
	})
}

// MustHavePrivilege replicates responder.MustHavePrivilege, but supports the
// workflow handler structure's needs
func MustHavePrivilege(priv *user.Privilege, f HandlerFunc) HandlerFunc {
	return HandlerFunc(func(resp *responder.Responder, i *Issue) {
		if resp.Vars.User.PermittedTo(priv) {
			f(resp, i)
			return
		}

		resp.Vars.Alert = "Insufficient Privileges"
		resp.Writer.WriteHeader(http.StatusForbidden)
		resp.Render(responder.InsufficientPrivileges)
	})
}

// canView verifies user can view metadata workflow information
func canView(h HandlerFunc) HandlerFunc {
	return MustHavePrivilege(user.ViewMetadataWorkflow, h)
}

// canWrite verifies user can enter metadata for an issue
func canWrite(h HandlerFunc) HandlerFunc {
	return MustHavePrivilege(user.EnterIssueMetadata, h)
}

// canReview verifies user can review metadata for an issue
func canReview(h HandlerFunc) HandlerFunc {
	return MustHavePrivilege(user.ReviewIssueMetadata, h)
}

// _canPerformWorkflow verifies the issue's workflow is of a type on which the
// user can take action
func _canPerformWorkflow(u *user.User, i *Issue) bool {
	switch i.WorkflowStep {
	case schema.WSReadyForMetadataEntry:
		return u.PermittedTo(user.EnterIssueMetadata)

	case schema.WSAwaitingMetadataReview:
		return u.PermittedTo(user.ReviewIssueMetadata)
	}

	return false
}

// canClaim makes sure the issue can be claimed via _canClaim
func canClaim(h HandlerFunc) HandlerFunc {
	return HandlerFunc(func(resp *responder.Responder, i *Issue) {
		var u = resp.Vars.User
		if i.IsOwned() {
			logger.Warnf("User %s trying to perform an action on issue %d which is owned by user %d",
				u.Login, i.ID, i.WorkflowOwnerID)
			resp.Vars.Alert = "You cannot take action on this issue; it's been claimed by another user"
			resp.Writer.WriteHeader(http.StatusForbidden)
			resp.Render(responder.Empty)
			return
		}
		if i.WorkflowStep != schema.WSReadyForMetadataEntry && i.WorkflowStep != schema.WSAwaitingMetadataReview {
			logger.Warnf("User %s trying to claim issue %d which has workflow step %s",
				resp.Vars.User.Login, i.ID, i.WorkflowStepString)
			resp.Vars.Alert = "Error: invalid action for this issue"
			resp.Writer.WriteHeader(http.StatusBadRequest)
			resp.Render(responder.Empty)
			return
		}

		h(resp, i)
	})
}

// issueNeedsMetadataEntry verifies that the issue's workflow step is valid for
// entering metadata
func issueNeedsMetadataEntry(h HandlerFunc) HandlerFunc {
	return HandlerFunc(func(resp *responder.Responder, i *Issue) {
		if i.WorkflowStep != schema.WSReadyForMetadataEntry {
			logger.Warnf("User %s trying to perform a metadata entry action on issue %d which has workflow step %s",
				resp.Vars.User.Login, i.ID, i.WorkflowStepString)
			resp.Vars.Alert = "Error: invalid action for this issue"
			resp.Writer.WriteHeader(http.StatusBadRequest)
			resp.Render(responder.Empty)
			return
		}

		h(resp, i)
	})
}

// issueAwaitingMetadataReview verifies that the issue's workflow step is valid
// for reviewing metadata
func issueAwaitingMetadataReview(h HandlerFunc) HandlerFunc {
	return HandlerFunc(func(resp *responder.Responder, i *Issue) {
		if i.WorkflowStep != schema.WSAwaitingMetadataReview {
			logger.Warnf("User %s trying to perform a metadata review action on issue %d which has workflow step %s",
				resp.Vars.User.Login, i.ID, i.WorkflowStepString)
			resp.Vars.Alert = "Error: invalid action for this issue"
			resp.Writer.WriteHeader(http.StatusBadRequest)
			resp.Render(responder.Empty)
			return
		}

		h(resp, i)
	})
}

// ownsIssue doesn't allow a page hit unless the authenticated user is also the
// user who claimed the issue
func ownsIssue(h HandlerFunc) HandlerFunc {
	return HandlerFunc(func(resp *responder.Responder, i *Issue) {
		var u = resp.Vars.User
		if i.WorkflowOwnerID != u.ID {
			logger.Warnf("User %s trying to perform an action on unowned issue %d", u.Login, i.ID)
			resp.Vars.Alert = "You cannot take action on this issue; it is not claimed by you"
			resp.Writer.WriteHeader(http.StatusForbidden)
			resp.Render(responder.Empty)
			return
		}

		h(resp, i)
	})
}
