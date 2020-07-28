package workflowhandler

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
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

func saveUnfixableHandler(resp *responder.Responder, i *Issue) {
	resp.Vars.Alert = template.HTML("Error: not implemented")
	resp.Writer.WriteHeader(http.StatusInternalServerError)
	resp.Render(responder.Empty)
}
