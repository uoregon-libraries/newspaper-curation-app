package batchhandler

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

// setStatus centralizes the process of setting the status and handling the
// info/alert needed on success or error
func setStatus(r *Responder, status string, t *tmpl.Template) bool {
	var oldStatus = r.batch.Status
	r.batch.Status = status
	var err = r.batch.Save()
	if err != nil {
		// Since we're merely re-rending the template, we must put the batch back
		// to its original state or the template could be weird/broken
		r.batch.Status = oldStatus
		logger.Errorf("Unable to set batch %d (%s) status to %s: %s", r.batch.ID, r.batch.FullName(), status, err)
		r.Vars.Title = "Error saving batch"
		r.Vars.Alert = template.HTML("Unable to update batch status. Try again or contact support.")
		r.Render(t)
		return false
	}

	return true
}

// listHandler spits out the list of batches
func listHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Batches"
	var list, err = models.InProcessBatches()
	if err != nil {
		logger.Errorf("Unable to load batches: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull batch list - try again or contact support")
		return
	}

	r.Vars.Data["Batches"] = wrapBatches(list)
	r.Vars.Data["Can"] = Can(r.Vars.User)
	r.Render(listTmpl)
}

func viewHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.can.View(r.batch) {
		r.Error(http.StatusForbidden, "You are not permitted to view this batch")
		return
	}
	r.Vars.Title = fmt.Sprintf("Viewing batch (%s)", r.batch.Name)
	r.Render(viewTmpl)
}

func qcReadyHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.can.Load(r.batch) {
		r.Error(http.StatusForbidden, "You are not permitted to load batches or flag them for having been loaded")
		return
	}
	if !setStatus(r, models.BatchStatusQCReady, viewTmpl) {
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: r.batch.Name + ": status updated to QC Ready", Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}

func qcApproveFormHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.can.Approve(r.batch) {
		r.Error(http.StatusForbidden, "You are not permitted to approve this batch for a production load")
		return
	}

	r.Vars.Title = "Approve batch?"
	r.Render(approveFormTmpl)
}

func qcApproveHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.can.Approve(r.batch) {
		r.Error(http.StatusForbidden, "You are not permitted to approve this batch for a production load")
		return
	}
	if !setStatus(r, models.BatchStatusPassedQC, approveFormTmpl) {
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: r.batch.Name + ": approved for production load", Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}

func qcRejectFormHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.can.Reject(r.batch) {
		r.Error(http.StatusForbidden, "You are not permitted to reject this batch")
		return
	}

	r.Vars.Title = "Reject batch?"
	r.Render(rejectFormTmpl)
}

func qcRejectHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = getBatchResponder(w, req)
	if !ok {
		return
	}
	if !r.can.Reject(r.batch) {
		r.Error(http.StatusForbidden, "You are not permitted to reject this batch")
		return
	}

	if !setStatus(r, models.BatchStatusQCFlagIssues, rejectFormTmpl) {
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: r.batch.Name + ": failed QC, ready to flag issues for removal", Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}
