package workflowhandler

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// enterErrorHandler displays the form to enter an error for the given issue
func enterErrorHandler(resp *responder.Responder, i *Issue) {
	resp.Vars.Title = "Report Issue Error"
	resp.Vars.Data["Issue"] = i
	resp.Render(ReportErrorTmpl)
}

// saveErrorHandler records the error in the database, unclaims the issue, and
// flags it as needing admin attention
func saveErrorHandler(resp *responder.Responder, i *Issue) {
	var emsg = resp.Request.FormValue("error")
	if emsg == "" {
		http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Error report empty; no action taken", Path: "/"})
		http.Redirect(resp.Writer, resp.Request, i.Path("metadata"), http.StatusFound)
		return
	}

	var err = i.ReportError(resp.Vars.User.ID, emsg)
	if err != nil {
		logger.Errorf("Unable to save issue id %d's error (POST: %#v): %s", i.ID, resp.Request.Form, err)
		resp.Vars.Alert = template.HTML("Error trying to save error report (no, the irony is not lost on us); try again or contact support")
		resp.Writer.WriteHeader(http.StatusInternalServerError)
		resp.Render(responder.Empty)
		return
	}

	resp.Audit("report-error", fmt.Sprintf("issue id %d", i.ID))
	http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Issue error reported", Path: "/"})
	http.Redirect(resp.Writer, resp.Request, basePath, http.StatusFound)
}

func reviewUnfixableHandler(resp *responder.Responder, i *Issue) {
	resp.Vars.Title = "Reviewing Issue Error(s)"
	resp.Vars.Data["Issue"] = i
	resp.Render(ViewErrorTmpl)
}

func viewReturnUnfixableFormHandler(resp *responder.Responder, i *Issue) {
	var list, err = models.ActiveUsers()
	if err != nil {
		logger.Errorf("Unable to retrieve user list for unfixable form handler: %s", err)
		resp.Vars.Alert = template.HTML("Error trying to return this issue to NCA; try again or contact support")
		resp.Writer.WriteHeader(http.StatusInternalServerError)
		resp.Render(responder.Empty)
		return
	}

	resp.Vars.Title = "Return issue to NCA workflow"
	resp.Vars.Data["Issue"] = i
	resp.Vars.Data["Users"] = list
	resp.Render(ReturnIssueToNCATmpl)
}

func viewRemoveUnfixableFormHandler(resp *responder.Responder, i *Issue) {
	resp.Vars.Title = "Remove issue from NCA"
	resp.Vars.Data["Issue"] = i
	resp.Render(RemoveIssueFromNCATmpl)
}

func returnErrorIssueHandler(resp *responder.Responder, i *Issue) {
	var action = resp.Request.FormValue("action")
	var comment = resp.Request.FormValue("comment")
	var wID, _ = strconv.Atoi(resp.Request.FormValue("workflow_owner_id"))
	var err error

	switch action {
	case "return-to-entry":
		err = i.Issue.ReturnForCuration(resp.Vars.User.ID, wID, comment)
	case "return-to-review":
		err = i.Issue.ReturnForReview(resp.Vars.User.ID, wID, comment)
	}

	if err != nil {
		logger.Errorf("Unable to return errored issue (id %d, POST: %#v): %s", i.ID, resp.Request.Form, err)
		resp.Vars.Alert = template.HTML("Error trying to return this issue to NCA; try again or contact support")
		resp.Writer.WriteHeader(http.StatusInternalServerError)
		resp.Render(responder.Empty)
		return
	}

	resp.Audit("undo-error-issue", fmt.Sprintf("issue %d %s, comment: %q", i.ID, action, comment))
	http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Issue moved back to NCA successfully", Path: "/"})
	http.Redirect(resp.Writer, resp.Request, basePath, http.StatusFound)
}

func removeUnfixableIssueHandler(resp *responder.Responder, i *Issue) {
	var comment = resp.Request.FormValue("comment")
	jobs.QueueRemoveErroredIssue(i.Issue, conf.ErroredIssuesPath)

	resp.Audit("remove-error-issue", fmt.Sprintf("issue %d, comment: %q", i.ID, comment))
	http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Issue is now being moved to the error folder", Path: "/"})
	http.Redirect(resp.Writer, resp.Request, basePath, http.StatusFound)
}
