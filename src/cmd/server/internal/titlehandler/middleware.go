package titlehandler

import (
	"net/http"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db/user"
)

// canView verifies the user can view the user list
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ListTitles, h)
}

// canModify verifies the user can create/edit/delete users
func canModify(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ModifyTitles, h)
}
