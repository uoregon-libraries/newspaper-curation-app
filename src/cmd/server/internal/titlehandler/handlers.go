package titlehandler

import (
	"cmd/server/internal/responder"
	"config"
	"db"
	"net/http"
	"path"
	"strconv"
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

	// formTmpl is the form for adding or editing a title
	formTmpl *tmpl.Template
)

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config) {
	conf = c
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(listHandler))
	s.Path("/new").Handler(canModify(newHandler))
	s.Path("/edit").Handler(canModify(editHandler))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{
		"TitlesHomeURL": func() string { return basePath },
	})
	layout.Path = path.Join(layout.Path, "titles")

	listTmpl = layout.MustBuild("list.go.html")
	formTmpl = layout.MustBuild("form.go.html")
}

func getTitle(r *responder.Responder) (t *Title, handled bool) {
	var idStr = r.Request.FormValue("id")
	var id, _ = strconv.Atoi(idStr)
	if id < 1 {
		logger.Warnf("Invalid title id for request %q (%s)", r.Request.URL.Path, idStr)
		r.Error(http.StatusBadRequest, "Invalid title id - try again or contact support")
		return nil, true
	}

	var dbt, err = db.FindTitleByID(id)
	if err != nil {
		logger.Errorf("Unable to find title by id %d: %s", id, err)
		r.Error(http.StatusInternalServerError, "Unable to find title - try again or contact support")
		return nil, true
	}
	if dbt == nil {
		r.Error(http.StatusNotFound, "Unable to find title - try again or contact support")
		return nil, true
	}

	return WrapTitle(dbt), false
}

// listHandler spits out the list of titles
func listHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Titles"
	var dbTitles, err = db.Titles()
	if err != nil {
		logger.Errorf("Unable to load title list: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull title list - try again or contact support")
		return
	}

	var titles = WrapTitles(dbTitles)
	SortTitles(titles)
	r.Vars.Data["Titles"] = titles
	r.Render(listTmpl)
}

// newHandler shows a form for adding a new title
func newHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Data["Title"] = WrapTitle(&db.Title{})
	r.Vars.Title = "Creating a new title"
	r.Render(formTmpl)
}

// editHandler loads the title by id and renders the edit form
func editHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var t, handled = getTitle(r)
	if handled {
		return
	}

	r.Vars.Data["Title"] = t
	r.Vars.Title = "Editing " + t.Name
	r.Render(formTmpl)
}
