package uploadedissuehandler

import (
	"cmd/server/internal/responder"
	"net/http"
	"user"
)

// canView verifies the user can view uploaded issues
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ViewUploadedIssues, h)
}

// canModify verifies the user can queue uploaded issues to get them into the
// workflow (and eventually maybe "pre-reject" issues and alert somebody?)
func canModify(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ModifyUploadedIssues, h)
}
