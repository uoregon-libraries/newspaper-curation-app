package mochandler

import (
	"cmd/server/internal/responder"
	"config"
	"db"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strconv"
	"user"
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

	// listTmpl is the template which shows all MOCs and the add/remove options
	listTmpl *tmpl.Template

	// formTmpl is the form for adding a new MOC
	formTmpl *tmpl.Template
)

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config) {
	conf = c
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(listHandler))
	s.Path("/new").Handler(canAdd(newHandler))
	s.Path("/save").Methods("POST").Handler(canAdd(saveHandler))
	s.Path("/delete").Methods("POST").Handler(canDelete(deleteHandler))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{"MOCHomeURL": func() string { return basePath }})
	layout.Path = path.Join(layout.Path, "mocs")

	listTmpl = layout.MustBuild("list.go.html")
	formTmpl = layout.MustBuild("form.go.html")
}

// listHandler spits out the list of MOCs
func listHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	var r = responder.Response(w, req)
	r.Vars.Title = "MARC Org Code List"
	r.Vars.Data["MOCs"], err = db.AllMOCs()
	if err != nil {
		logger.Errorf("Unable to load MOC list: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull MOC list - try again or contact support")
		return
	}
	r.Render(listTmpl)
}

// newHandler shows a form for adding a new MOC
func newHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Create a new MARC Org Code"
	r.Render(formTmpl)
}

// saveHandler writes the new MOC to the db
func saveHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var code = req.FormValue("code")
	if db.ValidMOC(code) {
		r.Vars.Alert = template.HTML(fmt.Sprintf("MOC %q already exists", code))
		r.Render(formTmpl)
		return
	}

	var moc, err = db.CreateMOC(code)
	if err != nil {
		logger.Errorf("Unable to create new MOC %q: %s", moc, err)
		r.Error(http.StatusInternalServerError, "Error trying to create new MOC - try again or contact support")
		return
	}

	r.Audit("create-moc", code)
	http.SetCookie(w, &http.Cookie{Name: "Info", Value: "New MOC created", Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}

// deleteHandler removes the given MOC from the db
func deleteHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var idString = req.FormValue("id")
	var id, _ = strconv.Atoi(idString)
	if id < 1 {
		logger.Errorf("Invalid MOC id to delete (%s)", idString)
		r.Error(http.StatusInternalServerError, "Error trying to delete MOC - try again or contact support")
		return
	}

	var err = db.DeleteMOC(id)
	if err != nil {
		logger.Errorf("Unable to delete MOC (id %d): %s", id, err)
		r.Error(http.StatusInternalServerError, "Error trying to delete MOC - try again or contact support")
		return
	}

	r.Audit("delete-moc", idString)
	http.SetCookie(w, &http.Cookie{Name: "Info", Value: "Deleted MOC", Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}

// canView verifies the user can view MOCs - right now this just checks a
// single MOC permission, but we're splitting it out just in case that changes
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ManageMOCs, h)
}

// canAdd verifies the user can create new MOCs - right now this just checks a
// single MOC permission, but we're splitting it out just in case that changes
func canAdd(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ManageMOCs, h)
}

// canDelete verifies the user can create new MOCs - right now this just checks
// a single MOC permission, but we're splitting it out just in case that changes
func canDelete(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(user.ManageMOCs, h)
}
