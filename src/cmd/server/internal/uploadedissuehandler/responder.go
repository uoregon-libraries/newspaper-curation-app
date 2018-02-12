package uploadedissuehandler

import (
	"cmd/server/internal/responder"
	"fmt"
	"net/http"
	"web/tmpl"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/gopkg/logger"
)

type respError struct {
	status int
	msg    string
}

// resp wraps responder.Responder to add in some data that is useful to
// auto-load in all uploaded issue handling
type resp struct {
	*responder.Responder
	sftpTitles []*Title
	title      *Title
	issue      *Issue
	vars       map[string]string
	err        *respError
}

// getResponder sets up a resp with default values for issue/title to avoid
// panics, and then attempts to load data based on vars
func getResponder(w http.ResponseWriter, req *http.Request) *resp {
	var r = &resp{
		Responder: responder.Response(w, req),
		title:     &Title{},
		issue:     &Issue{},
		vars:      mux.Vars(req),
	}
	r.loadTitles()
	r.loadTitle()
	r.loadIssue()

	return r
}

func (r *resp) loadTitles() {
	var err error
	r.sftpTitles, err = searcher.Titles()
	if err != nil {
		logger.Errorf("Couldn't load SFTP titles: %s", err)
		r.err = &respError{http.StatusInternalServerError, "Error trying to load titles; try again or contact support"}
	}
}

func (r *resp) loadTitle() {
	// If there's no "lccn" var, we don't expect (or look for) a title
	var lccn, ok = r.vars["lccn"]
	if !ok {
		return
	}

	// If we have an lccn var, it's an error to not find a title
	r.title = searcher.TitleLookup(lccn)
	if r.title == nil {
		r.err = &respError{http.StatusNotFound, fmt.Sprintf("Unable to find title %#v", lccn)}
	}
}

func (r *resp) loadIssue() {
	// Make sure we have a title, otherwise there's nothing to do here
	if r.title == nil {
		return
	}

	// If there's no "issue" var, there's also nothing to do here
	var issueDate, ok = r.vars["issue"]
	if !ok {
		return
	}

	r.issue = r.title.IssueLookup[issueDate]
	if r.issue == nil {
		var msg = fmt.Sprintf("Unable to find issue %#v for title %#v", issueDate, r.title.Name)
		r.err = &respError{http.StatusNotFound, msg}
	}
}

// Render sets up the titles/title/issue data vars for the template, then
// delegates to the base responder.Responder
func (r *resp) Render(t *tmpl.Template) {
	// Avoid any further work if we had an error
	if r.err != nil {
		r.Error(r.err.status, r.err.msg)
		return
	}

	// Set up all the data vars
	r.Vars.Data["Titles"] = r.sftpTitles
	r.Vars.Data["Title"] = r.title
	r.Vars.Data["Issue"] = r.issue

	r.Responder.Render(t)
}
