package batchmakerhandler

import (
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

var (
	basePath string

	// layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	layout *tmpl.TRoot

	// buildBatchFormTmpl is the form for finding issues to put into one or more batches
	buildBatchFormTmpl *tmpl.Template
)

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string) {
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canBuild(buildBatchForm))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{
		"BatchMakerHomeURL":   func() string { return basePath },
		"BatchMakerFilterURL": func() string { return path.Join(basePath, "filter") },
	})
	layout.Path = path.Join(layout.Path, "batchmaker")

	buildBatchFormTmpl = layout.MustBuild("build.go.html")
}

// buildBatchForm shows a form for filtering issues that are ready for batching
func buildBatchForm(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Filter Issues For Batching"
	r.Render(buildBatchFormTmpl)
}
