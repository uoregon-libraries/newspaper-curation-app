package uploadedissuehandler

import (
	"cmd/server/internal/responder"
	"fmt"
	"net/http"
	"web/tmpl"

	"github.com/gorilla/mux"
)

type respError struct {
	status int
	msg    string
}

// resp wraps responder.Responder to add in some data that is useful to
// auto-load in all uploaded issue handling
type resp struct {
	*responder.Responder
	bornDigitalTitles []*Title
	scannedTitles     []*Title
	title             *Title
	issue             *Issue
	vars              map[string]string
	err               *respError
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
	for _, t := range searcher.Titles() {
		switch t.Type {
		case TitleTypeScanned:
			r.scannedTitles = append(r.scannedTitles, t)
		case TitleTypeBornDigital:
			r.bornDigitalTitles = append(r.bornDigitalTitles, t)
		}
	}
}

func (r *resp) loadTitle() {
	// If there's no "title" var, we don't expect (or look for) a title
	var slug, ok = r.vars["title"]
	if !ok {
		return
	}

	// If we have a title var, it's an error to not find a title
	r.title = searcher.TitleLookup(slug)
	if r.title == nil {
		r.err = &respError{http.StatusNotFound, fmt.Sprintf("Unable to find title %#v", slug)}
	}
}

func (r *resp) loadIssue() {
	// Make sure we have a title, otherwise there's nothing to do here
	if r.title == nil {
		return
	}

	// If there's no "issue" var, there's also nothing to do here
	var dateEdition, ok = r.vars["issue"]
	if !ok {
		return
	}

	r.issue = r.title.IssueLookup[dateEdition]
	if r.issue == nil {
		var msg = fmt.Sprintf("Unable to find issue %#v for title %#v", dateEdition, r.title.Name)
		r.err = &respError{http.StatusNotFound, msg}
	}
}

// Render sets up the titles/title/issue data vars for the template, then
// delegates to the base responder.Responder
func (r *resp) Render(t *tmpl.Template) {
	// Hack in an error if the searcher has failed too often
	if searcher.FailedSearch() {
		r.err = &respError{http.StatusInternalServerError, "Unable to load titles and issues; try again or contact support"}
	}

	// Avoid any further work if we had an error
	if r.err != nil {
		r.Error(r.err.status, r.err.msg)
		return
	}

	// Set up all the data vars
	r.Vars.Data["BornDigitalTitles"] = r.bornDigitalTitles
	r.Vars.Data["ScannedTitles"] = r.scannedTitles
	r.Vars.Data["Title"] = r.title
	r.Vars.Data["Issue"] = r.issue

	r.Responder.Render(t)
}
