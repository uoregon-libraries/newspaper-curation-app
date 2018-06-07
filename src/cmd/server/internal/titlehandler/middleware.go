package titlehandler

import (
	"cmd/server/internal/responder"
	"db/user"
	"net/http"
)

// canView verifies the user can view the user list
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ListTitles, h)
}
