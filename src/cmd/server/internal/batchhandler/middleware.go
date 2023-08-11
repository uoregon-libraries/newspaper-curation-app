// middleware.go houses central middleware logic for batch management. We don't
// worry about super-granular auth checks like we do for the workflow handler,
// as that became a bit of a mess, so the batch-specific validations (e.g.,
// "can a user approve this *specific* batch" as opposed to "does the user have
// permissions to approve batches awaiting QC") happen in the handlers
// themselves.

package batchhandler

import (
	"net/http"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
)

// canView ensures a user is allowed to view batch status data
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ViewBatchStatus, h)
}

func canApprove(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ApproveQCReadyBatches, h)
}

func canReject(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.RejectQCReadyBatches, h)
}

func canArchive(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ArchiveBatches, h)
}
