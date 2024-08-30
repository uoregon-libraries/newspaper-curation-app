package batchmakerhandler

import (
	"net/http"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
)

// canBuild verifies the user can build a batch
func canBuild(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.GenerateBatches, h)
}
