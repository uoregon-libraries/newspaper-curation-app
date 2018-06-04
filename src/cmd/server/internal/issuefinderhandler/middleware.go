package issuefinderhandler

import (
	"cmd/server/internal/responder"
	"db/user"
	"net/http"
)

// canSearch verifies the user can search issues
func canSearch(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.SearchIssues, h)
}
