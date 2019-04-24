package issuefinderhandler

import (
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

var (
	basePath string
	watcher  *issuewatcher.Watcher
	conf     *config.Config

	// Layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	Layout *tmpl.TRoot

	// Tmpl renders the "find issues" form and search results in one page
	Tmpl *tmpl.Template
)

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config, w *issuewatcher.Watcher) {
	conf = c
	watcher = w
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canSearch(FormHandler))
	s.Path("/search").Handler(canSearch(ResultsHandler))

	Layout = responder.Layout.Clone()
	Layout.Path = path.Join(Layout.Path, "issuefinder")
	Tmpl = Layout.MustBuild("tmpl.go.html")
}

// FormHandler spits out the search form
func FormHandler(w http.ResponseWriter, req *http.Request) {
	var r = getResponder(w, req)
	r.Vars.Title = "Find Issues"
	r.Render(Tmpl)
}

// ResultsHandler spits out the search form as well as the results for the
// given search.  Or errors, if the search was invalid in any way.
func ResultsHandler(w http.ResponseWriter, req *http.Request) {
	var r = getResponder(w, req)
	if len(r.Issues) == 0 {
		r.Vars.Title = "No Results"
	} else {
		r.Vars.Title = "Search Results"
	}
	r.Render(Tmpl)
}
