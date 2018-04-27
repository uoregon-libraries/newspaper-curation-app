package issuefinderhandler

import (
	"cmd/server/internal/responder"
	"net/http"
	"user"
)

// canSearch verifies the user can search issues
func canSearch(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.SearchIssues, h)
}
