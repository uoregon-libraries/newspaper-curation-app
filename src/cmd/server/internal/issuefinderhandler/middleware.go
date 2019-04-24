package issuefinderhandler

import (
	"net/http"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db/user"
)

// canSearch verifies the user can search issues
func canSearch(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.SearchIssues, h)
}
