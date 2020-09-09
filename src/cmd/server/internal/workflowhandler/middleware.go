package workflowhandler

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
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

			var i, err = models.FindIssue(id)
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
func MustHavePrivilege(priv *privilege.Privilege, f HandlerFunc) HandlerFunc {
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
	return MustHavePrivilege(privilege.ViewMetadataWorkflow, h)
}

func canHandler(h HandlerFunc, canFunc func(*CanValidation, *Issue)) HandlerFunc {
	return HandlerFunc(func(resp *responder.Responder, i *Issue) {
		var can = Can(resp.Vars.User)
		canFunc(can, i)
		can.sendResponse(h, resp, i)
	})
}

func canClaim(h HandlerFunc) HandlerFunc {
	return canHandler(h, func(can *CanValidation, i *Issue) { can.Claim(i) })
}
func canUnclaim(h HandlerFunc) HandlerFunc {
	return canHandler(h, func(can *CanValidation, i *Issue) { can.Unclaim(i) })
}
func canEnterMetadata(h HandlerFunc) HandlerFunc {
	return canHandler(h, func(can *CanValidation, i *Issue) { can.EnterMetadata(i) })
}
func canReviewMetadata(h HandlerFunc) HandlerFunc {
	return canHandler(h, func(can *CanValidation, i *Issue) { can.ReviewMetadata(i) })
}
func canReviewUnfixable(h HandlerFunc) HandlerFunc {
	return canHandler(h, func(can *CanValidation, i *Issue) { can.ReviewUnfixable(i) })
}
