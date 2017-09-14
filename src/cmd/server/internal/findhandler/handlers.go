package findhandler

import (
	"cmd/server/internal/responder"
	"issuesearch"
	"legacyfinder"
	"net/http"
	"path"
	"strconv"
	"web/tmpl"

	"github.com/gorilla/mux"
)

var (
	// basePath is the path to the search handler.  If we add other search
	// sub-pages, they'll start with this path.
	basePath string
	watcher  *legacyfinder.Watcher

	// Layout represents a cloned version of the responder's Layout template for
	// building the issue finder's "sub-templates"
	Layout *tmpl.TRoot

	// ResultsTmpl is our search results page's template
	ResultsTmpl *tmpl.Template

	// SearchFormAction is the path used in the HTML form for hitting a search page
	SearchFormAction string
)

// Setup sets up all the handler-specific routing, templates, etc
func Setup(r *mux.Router, webPath string, w *legacyfinder.Watcher) {
	basePath = webPath
	SearchFormAction = basePath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(responder.CanSearchIssues(HomeHandler))

	watcher = w
	Layout = responder.Layout.Clone()
	Layout.Path = path.Join(Layout.Path, "find")
	ResultsTmpl = Layout.MustBuild("results.go.html")
}

// rsp returns a Response pre-populated with data vars specific to this handler
func rsp(w http.ResponseWriter, req *http.Request) *responder.Responder {
	var r = responder.Response(w, req)
	r.Vars.Data["SearchFormAction"] = SearchFormAction
	return r
}

// assignUniqueTitles puts a title list into the given responder's data
func assignUniqueTitles(r *responder.Responder) {
	var titles = watcher.IssueFinder().Titles.Unique()
	titles.SortByName()
	r.Vars.Data["Titles"] = titles
}

// HomeHandler shows the search form and results if a search was performed
func HomeHandler(w http.ResponseWriter, req *http.Request) {
	var r = rsp(w, req)
	assignUniqueTitles(r)

	r.Vars.Title = "Find Issues"

	var lccn = req.FormValue("lccn")
	var year, _ = strconv.Atoi(req.FormValue("year"))
	var month, _ = strconv.Atoi(req.FormValue("month"))
	var day, _ = strconv.Atoi(req.FormValue("day"))
	r.Vars.Data["LCCN"] = lccn
	r.Vars.Data["Year"] = year
	r.Vars.Data["Month"] = month
	r.Vars.Data["Day"] = day

	if lccn == "" {
		r.Vars.Data["Issues"] = []*Issue{}
		r.Vars.Data["LCCN"] = ""
		r.Render(ResultsTmpl)
		return
	}

	var key, err = issuesearch.NewKey(lccn, year, month, day, 0)
	if err == nil {
		r.Vars.Data["Issues"] = getIssues(key)
	} else {
		r.Vars.Alert = "Invalid date value: " + err.Error()
	}
	r.Render(ResultsTmpl)
}

func getIssues(k *issuesearch.Key) []*Issue {
	var schemaIssues = watcher.LookupIssues(k)
	var issues = make([]*Issue, len(schemaIssues))
	for i, issue := range schemaIssues {
		issues[i] = &Issue{Issue: issue}
		issues[i].Namespace = watcher.IssueFinder().IssueNamespace[issue]
	}
	return issues
}
