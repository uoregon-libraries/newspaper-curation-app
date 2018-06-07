package titlehandler

import (
	"cmd/server/internal/responder"
	"config"
	"db"
	"net/http"
	"path"
	"web/tmpl"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/gopkg/logger"
)

var (
	basePath string
	conf     *config.Config

	// layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	layout *tmpl.TRoot

	// listTmpl is the template which shows all titles
	listTmpl *tmpl.Template
)

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config) {
	conf = c
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(listHandler))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{
		"TitlesHomeURL": func() string { return basePath },
	})
	layout.Path = path.Join(layout.Path, "titles")

	listTmpl = layout.MustBuild("list.go.html")
}

// listHandler spits out the list of titles
func listHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Titles"
	var titles, err = db.Titles()
	if err != nil {
		logger.Errorf("Unable to load title list: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull title list - try again or contact support")
		return
	}

	r.Vars.Data["Titles"] = titles
	r.Render(listTmpl)
}
