package mochandler

import (
	"net/http"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
)

// canView verifies the user can view MOCs - right now this just checks a
// single MOC permission, but we're splitting it out just in case that changes
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ManageMOCs, h)
}

// canAdd verifies the user can create new MOCs - right now this just checks a
// single MOC permission, but we're splitting it out just in case that changes
func canAdd(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ManageMOCs, h)
}

// canAdd verifies the user can edit MOCs - right now this just checks a single
// MOC permission, but we're splitting it out just in case that changes
func canEdit(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ManageMOCs, h)
}

// canDelete verifies the user can create new MOCs - right now this just checks
// a single MOC permission, but we're splitting it out just in case that changes
func canDelete(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ManageMOCs, h)
}
