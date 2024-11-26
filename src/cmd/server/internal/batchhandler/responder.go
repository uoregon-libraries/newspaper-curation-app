package batchhandler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// Responder wraps the central Responder to add custom data we require for most
// handlers related to batch processing
type Responder struct {
	*responder.Responder
	batch *Batch
}

// getBatchResponder centralizes the most common handler logic where we require
// a valid batch id in the request, and the flagged and normal issues
// associated with the batch
func getBatchResponder(w http.ResponseWriter, req *http.Request) (r *Responder, ok bool) {
	r = &Responder{Responder: responder.Response(w, req)}
	var idStr = mux.Vars(req)["batch_id"]
	var id, err = strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		r.Error(http.StatusBadRequest, fmt.Sprintf("Error: %q is not a valid batch id; check your URL and try again", idStr))
		return r, false
	}
	var b *models.Batch
	b, err = models.FindBatch(id)
	if err != nil {
		logger.Criticalf("Unable to load batch %d: %s", id, err)
		r.Error(http.StatusInternalServerError, "Error loading batch - try again or contact support")
		return r, false
	}
	if b == nil {
		r.Error(http.StatusNotFound, fmt.Sprintf("Batch %d does not exist - it may have been removed from NCA or otherwise made unavailable since you last viewed the batch list. Check your URL and try again or return to the batch list", id))
		return r, false
	}

	r.batch, err = wrapBatch(b, r.Vars.User)
	if err != nil {
		logger.Criticalf("Error reading flagged issues for batch %d (%s): %s", r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Error trying to read batch's issues - try again or contact support")
		return r, false
	}

	r.Vars.Data["Actions"] = r.batch.Actions
	r.Vars.Data["FlaggedIssues"] = r.batch.FlaggedIssues
	r.Vars.Data["UnflaggedIssues"] = r.batch.UnflaggedIssues
	r.Vars.Data["Batch"] = r.batch
	return r, true
}
