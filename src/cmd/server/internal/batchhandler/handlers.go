package batchhandler

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

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
		logger.Criticalf("Unable to set batch %d (%s) status to %s: %s", r.batch.ID, r.batch.FullName(), status, err)
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
		logger.Criticalf("Unable to load batches: %s", err)
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

func prepFlagging(w http.ResponseWriter, req *http.Request) (r *Responder, ok bool) {
	r, ok = getBatchResponder(w, req)
	if !ok {
		return r, false
	}
	if !r.can.FlagIssues(r.batch) {
		r.Error(http.StatusForbidden, "You are not permitted to flag issues for removal from this batch")
		return r, false
	}

	var err error
	r.Vars.Data["FlaggedIssues"], err = r.batch.FlaggedIssues()
	if err != nil {
		logger.Criticalf("Error reading flagged issues for batch %d (%s): %s", r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Error trying to read batch's issues - try again or contact support")
		return r, false
	}

	r.Vars.Title = "Rejecting batch"
	return r, true
}

func qcFlagIssuesFormHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = prepFlagging(w, req)
	if ok {
		r.Render(flagIssuesFormTmpl)
	}
}

func parseIssueKeyURL(val string) (string, error) {
	var u, err = url.Parse(val)
	if err != nil {
		return "", fmt.Errorf("%q is not a URL: %s", val, err)
	}
	var parts = strings.Split(u.Path, "/")
	for i, part := range parts {
		if part == "lccn" {
			if len(parts) < i+4 {
				return "", fmt.Errorf("%q is not a full URL to an issue", val)
			}
			var ed = parts[i+3]
			if !strings.HasPrefix(ed, "ed-") || len(ed) < 4 {
				return "", fmt.Errorf("%q doesn't have a valid edition", val)
			}
			ed = ed[3:]
			if len(ed) == 1 {
				ed = "0" + ed
			}
			val = parts[i+1] + "/" + strings.Replace(parts[i+2], "-", "", 2) + ed
			return val, nil
		}
	}
	return "", fmt.Errorf("%q is not a valid issue URL", val)
}

func parseIssueKeyStd(val string) (string, error) {
	// Issue keys must be exactly 21 characters: 10 for LCCN, slash, 10 for date
	// + edition
	if len(val) < 21 {
		return "", fmt.Errorf("%q is too short", val)
	}
	if len(val) > 21 {
		return "", fmt.Errorf("%q is too long", val)
	}
	var parts = strings.Split(val, "/")
	var lccn, dte string
	if len(parts) == 2 {
		lccn, dte = parts[0], parts[1]
	}
	if len(lccn) != 10 || len(dte) != 10 {
		return "", fmt.Errorf("%q is not an issue key", val)
	}

	var dt = dte[:8]
	var _, err = time.Parse("20060102", dt)
	if err != nil {
		return "", fmt.Errorf("%q is not a valid issue key: date part (%q) is not a real date", val, dt)
	}

	return val, nil
}

func qcFlagIssuesHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = prepFlagging(w, req)
	if !ok {
		return
	}

	req.ParseForm()
	var key = req.Form.Get("issue-key")
	var desc = req.Form.Get("issue-desc")

	// In just about every case where we render the template rather than
	// redirect, we need the following things set up
	r.Vars.Data["IssueKey"] = key
	r.Vars.Data["IssueDescription"] = desc

	var err error
	var errAlert string
	var showURLHelp, showKeyHelp bool
	if len(key) > 4 && key[:4] == "http" {
		errAlert = "Invalid issue URL"
		showURLHelp = true
		key, err = parseIssueKeyURL(key)
	} else {
		errAlert = "Invalid issue key"
		showKeyHelp = true
		key, err = parseIssueKeyStd(key)
	}
	if err != nil {
		r.Vars.Title = "Error - Rejecting batch"
		r.Vars.Alert = template.HTML(errAlert + ": " + err.Error())
		r.Vars.Data["ShowURLHelp"] = showURLHelp
		r.Vars.Data["ShowKeyHelp"] = showKeyHelp
		r.Render(flagIssuesFormTmpl)
		return
	}

	// Find issue and add it to the removal queue
	var i *models.Issue
	i, err = models.FindIssueByKey(key)
	if err != nil {
		logger.Criticalf("Error adding issue %q to batch %d (%s) for removal: %s", key, r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Database error trying to reject the issue. Try again or contact support.")
		return
	}
	if i == nil {
		r.Vars.Title = "Issue not found - Rejecting batch"
		r.Vars.Alert = template.HTML(errAlert + ": no such issue exists. Double-check your input and try again.")
		r.Vars.Data["ShowURLHelp"] = showURLHelp
		r.Vars.Data["ShowKeyHelp"] = showKeyHelp
		r.Render(flagIssuesFormTmpl)
		return
	}
	if i.BatchID != r.batch.ID {
		r.Vars.Title = "Error - Rejecting batch"
		r.Vars.Alert = template.HTML(fmt.Sprintf("%s: an issue matches your entry, but it is not part of batch %s. Double-check your input and try again.", errAlert, r.batch.Name))
		r.Render(flagIssuesFormTmpl)
		return
	}

	err = r.batch.FlagIssue(i, r.Vars.User, desc)
	if err != nil {
		logger.Criticalf("Error adding issue %q to batch %d (%s) for removal: %s", key, r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Database error trying to reject the issue. Try again or contact support.")
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: fmt.Sprintf("Flagged issue %s for removal", i.Key()), Path: "/"})
	http.Redirect(w, req, flagIssuesURL(r.batch), http.StatusFound)
}
