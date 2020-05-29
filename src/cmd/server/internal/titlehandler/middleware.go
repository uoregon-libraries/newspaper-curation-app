package titlehandler

import (
	"net/http"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
)

// canView verifies the user can view the titles list
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ListTitles, h)
}

// canModify verifies the user can create/edit/delete titles
func canModify(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ModifyTitles, h)
}
