package batchmakerhandler

import (
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
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

	var aggs, err = models.MOCIssueAggregations()
	if err != nil {
		logger.Errorf("Unable to load MOC issue aggregation: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull MOC list - try again or contact support")
		return
	}

	// Remove information that won't help anybody make decisions (e.g., InProduction)
	for _, agg := range aggs {
		delete(agg.Counts, schema.WSNil)
		delete(agg.Counts, schema.WSInProduction)
	}

	r.Vars.Data["MOCIssueAggregations"] = aggs
	r.Render(buildBatchFormTmpl)
}
