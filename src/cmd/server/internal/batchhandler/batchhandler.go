package batchhandler

import (
	"net/http"
	"path"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

var (
	basePath string
	conf     *config.Config

	// layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	layout *tmpl.TRoot

	// listTmpl is the template which shows all batches and actions
	listTmpl *tmpl.Template

	// viewTmpl is the batch view for showing details about a batch
	viewTmpl *tmpl.Template
)

func batchURL(b *models.Batch, other ...string) string {
	var parts = []string{basePath, strconv.Itoa(b.ID)}
	if len(other) > 0 {
		parts = append(parts, other...)
	}
	return path.Join(parts...)
}

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config) {
	conf = c
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(listHandler))
	s.Path("/{batch_id}").Methods("GET").Handler(canView(viewHandler))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{
		"BatchesHomeURL": func() string { return basePath },
		"ViewURL":        func(b *models.Batch) string { return batchURL(b) },
	})
	layout.Path = path.Join(layout.Path, "batches")

	listTmpl = layout.MustBuild("list.go.html")
	viewTmpl = layout.MustBuild("view.go.html")
}

// listHandler spits out the list of batches
func listHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	var r = responder.Response(w, req)
	r.Vars.Title = "Batches"
	r.Vars.Data["Batches"], err = models.InProcessBatches()
	r.Vars.Data["Can"] = Can(r.Vars.User)
	if err != nil {
		logger.Errorf("Unable to load batches: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull batch list - try again or contact support")
		return
	}
	r.Render(listTmpl)
}

func viewHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Error(http.StatusInternalServerError, "Not implemented")
}
