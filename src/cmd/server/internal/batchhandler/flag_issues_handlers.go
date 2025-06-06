package batchhandler

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// flagIssuesHandler receives all flag-issue form POST requests, dispatching to
// the correct "sub-handler" based on the form's action
func flagIssuesHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = prepFlagging(w, req)
	if !ok {
		return
	}

	var err = req.ParseForm()
	if err != nil {
		logger.Errorf("Unable to read form in flagIssuesHandler: %s", err)
		r.Error(http.StatusInternalServerError, "Error processing submission. Try again or contact support.")
		return
	}

	switch req.Form.Get("action") {
	case "flag-issue":
		flagIssue(r)
	case "unflag-issue":
		unflagIssue(r)
	case "finalize":
		finalizeBatch(r)
	case "undo":
		undoBatch(r)
	case "abort":
		abortBatch(r)
	default:
		r.Error(http.StatusBadRequest, "Invalid request. Try again or contact support.")
	}
}

// prepFlagging runs the common logic for all handlers related to the flagging
// of issues after a batch has been marked for failure. It ensures we can get
// the batch responder, that the user is allowed to flag issues on the batch,
// and that we can read the already-flagged issues on the batch.
func prepFlagging(w http.ResponseWriter, req *http.Request) (r *Responder, ok bool) {
	r, ok = getBatchResponder(w, req)
	if !ok {
		return r, false
	}
	if !r.batch.Can().FlagIssues() {
		r.Error(http.StatusForbidden, "You are not permitted to flag issues for removal from this batch")
		return r, false
	}

	r.Vars.Data["RemainingIssues"] = len(r.batch.Issues) - len(r.batch.FlaggedIssues)
	r.Vars.Title = "Rejecting batch " + r.batch.Name
	return r, true
}

func flagIssuesFormHandler(w http.ResponseWriter, req *http.Request) {
	var r, ok = prepFlagging(w, req)
	if ok {
		r.Render(flagIssuesFormTmpl)
	}
}

// parseIssueKeyURL is a helper to validate that the URL value can be converted
// into an issue key, returning said key or an error if parsing fails.
func parseIssueKeyURL(val string) (string, error) {
	var u, err = url.Parse(val)
	if err != nil {
		return "", fmt.Errorf("%q is not a URL: %w", val, err)
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

// parseIssueKeyStd validates the format of val and that the date portion is a
// real date. val is returned as-is if parsing succeeds, otherwise a blank
// value and an error are returned.
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

func unflagIssue(r *Responder) {
	var id, _ = strconv.ParseInt(r.Request.Form.Get("issue-id"), 10, 64)
	if id < 1 {
		http.SetCookie(r.Writer, &http.Cookie{Name: "Alert", Value: "Invalid issue to unflag", Path: "/"})
		http.Redirect(r.Writer, r.Request, flagIssuesURL(r.batch), http.StatusBadRequest)
		return
	}

	var issue, err = models.FindIssue(id)
	if err != nil {
		logger.Errorf("Unable to look up issue %d: %s", id, err)
		r.Error(http.StatusInternalServerError, "Database error trying to unflag the issue. Try again or contact support.")
		return
	}
	if issue == nil {
		http.SetCookie(r.Writer, &http.Cookie{Name: "Alert", Value: "Unable to find issue to unflag. Try again or contact support.", Path: "/"})
		http.Redirect(r.Writer, r.Request, flagIssuesURL(r.batch), http.StatusNotFound)
		return
	}

	err = r.batch.UnflagIssue(issue)
	if err != nil {
		logger.Errorf("Unable to unflag issue %d for batch %d (%s): %s", id, r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Database error trying to unflag the issue. Try again or contact support.")
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: fmt.Sprintf("Took issue %s off the flagged-issue list", issue.Key()), Path: "/"})
	http.Redirect(r.Writer, r.Request, flagIssuesURL(r.batch), http.StatusFound)
}

func flagIssue(r *Responder) {
	var key = r.Request.Form.Get("issue-key")
	var desc = r.Request.Form.Get("issue-desc")

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
		r.Vars.Title = "Error - " + r.Vars.Title
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
		logger.Errorf("Error adding issue %q to batch %d (%s) for removal: %s", key, r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Database error trying to reject the issue. Try again or contact support.")
		return
	}
	if i == nil {
		r.Vars.Title = "Issue not found - " + r.Vars.Title
		r.Vars.Alert = template.HTML(errAlert + ": no such issue exists. Double-check your input and try again.")
		r.Vars.Data["ShowURLHelp"] = showURLHelp
		r.Vars.Data["ShowKeyHelp"] = showKeyHelp
		r.Render(flagIssuesFormTmpl)
		return
	}
	if i.BatchID != r.batch.ID {
		r.Vars.Title = "Error - " + r.Vars.Title
		r.Vars.Alert = template.HTML(fmt.Sprintf("%s: an issue matches your entry, but it is not part of batch %s. Double-check your input and try again.", errAlert, r.batch.Name))
		r.Render(flagIssuesFormTmpl)
		return
	}

	err = r.batch.FlagIssue(i, r.Vars.User, desc)
	if err != nil {
		logger.Errorf("Error adding issue %q to batch %d (%s) for removal: %s", key, r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Database error trying to reject the issue. Try again or contact support.")
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: fmt.Sprintf("Flagged issue %s for removal", i.Key()), Path: "/"})
	http.Redirect(r.Writer, r.Request, flagIssuesURL(r.batch), http.StatusFound)
}

func abortBatch(r *Responder) {
	var err = r.batch.AbortIssueFlagging(r.Vars.User)
	if err != nil {
		logger.Errorf("Unable to abort issue flagging for batch %d (%s): %s", r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Database error trying to reset the batch. Try again or contact support.")
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: fmt.Sprintf("Batch %q has been reset and is ready for QC again", r.batch.Name), Path: "/"})
	http.Redirect(r.Writer, r.Request, batchURL(r.batch), http.StatusFound)
}

func finalizeBatch(r *Responder) {
	var err = r.batch.Save(models.ActionTypeFinalizeBatch, r.Vars.User.ID, "")
	if err != nil {
		logger.Errorf(`Unable to log "finalize batch" action for batch %d (%s): %s`, r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Error trying to finalize the batch. Try again or contact support.")
		return
	}

	// If all issues were flagged for removal, we delete the batch entirely
	if len(r.batch.Issues) == len(r.batch.FlaggedIssues) {
		queueDeleteBatchJob(r)
		return
	}

	// There are enough moving pieces here that we have to queue this up in the
	// background rather than just run a quick DB operation or something
	err = jobs.QueueBatchFinalizeIssueFlagging(r.batch.Batch, r.batch.FlaggedIssues, conf)
	if err != nil {
		logger.Errorf("Unable to queue job to finalize issue flagging for batch %d (%s): %s", r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Error trying to finalize the batch. Try again or contact support.")
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: fmt.Sprintf("A background job has been queued to finalize batch %q", r.batch.Name), Path: "/"})
	http.Redirect(r.Writer, r.Request, basePath, http.StatusFound)
}

func undoBatch(r *Responder) {
	var err = r.batch.Save(models.ActionTypeUndoBatch, r.Vars.User.ID, "")
	if err != nil {
		logger.Errorf(`Unable to log "undo batch" action for batch %d (%s): %s`, r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Error trying to undo the batch. Try again or contact support.")
		return
	}

	queueDeleteBatchJob(r)
}

func queueDeleteBatchJob(r *Responder) {
	var err = jobs.QueueBatchForDeletion(r.batch.Batch, r.batch.FlaggedIssues, conf)
	if err != nil {
		logger.Errorf("Unable to queue job to delete batch %d (%s): %s", r.batch.ID, r.batch.Name, err)
		r.Error(http.StatusInternalServerError, "Error trying to finalize the batch. Try again or contact support.")
		return
	}

	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: fmt.Sprintf("A background job has been queued to finalize batch %q", r.batch.Name), Path: "/"})
	http.Redirect(r.Writer, r.Request, basePath, http.StatusFound)
}
