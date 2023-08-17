package mochandler

import (
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

var (
	basePath string

	// layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	layout *tmpl.TRoot

	// listTmpl is the template which shows all MOCs and the add/remove options
	listTmpl *tmpl.Template

	// formTmpl is the form for adding a new MOC
	formTmpl *tmpl.Template
)

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string) {
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(listHandler))
	s.Path("/new").Handler(canAdd(newHandler))
	s.Path("/edit").Handler(canEdit(editHandler))
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
	r.Vars.Data["MOCs"], err = models.AllMOCs()
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

	if r.Request.FormValue("id") != "" {
		updateMOC(r)
		return
	}

	createMOC(r)
}

func createMOC(r *responder.Responder) {
	var code = r.Request.FormValue("code")
	var name = r.Request.FormValue("name")
	if models.ValidMOC(code) {
		r.Vars.Alert = template.HTML(fmt.Sprintf("MOC %q already exists", code))
		r.Render(formTmpl)
		return
	}

	var moc = &models.MOC{Code: code, Name: name}
	var err = moc.Save()
	if err != nil {
		logger.Errorf("Unable to create new MOC %q: %s", moc, err)
		r.Error(http.StatusInternalServerError, "Error trying to create new MOC - try again or contact support")
		return
	}

	r.Audit(models.AuditActionCreateMoc, code)
	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: "New MOC created", Path: "/"})
	http.Redirect(r.Writer, r.Request, basePath, http.StatusFound)
}

func updateMOC(r *responder.Responder) {
	var moc, handled = getMOC(r)
	if handled {
		return
	}
	var oldMOC = &models.MOC{
		ID:   moc.ID,
		Code: moc.Code,
		Name: moc.Name,
	}
	var code = r.Request.FormValue("code")
	var name = r.Request.FormValue("name")
	moc.Code = code
	moc.Name = name
	var err = moc.Save()

	if err != nil {
		logger.Errorf("Unable to save MOC %q: %s", moc, err)
		r.Error(http.StatusInternalServerError, "Error trying to save MOC - try again or contact support")
		return
	}

	r.Audit(models.AuditActionUpdateMoc, fmt.Sprintf("previous: %#v, new: %#v", oldMOC, moc))
	http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: "MOC updated", Path: "/"})
	http.Redirect(r.Writer, r.Request, basePath, http.StatusFound)
}

// deleteHandler removes the given MOC from the db
func deleteHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var moc, handled = getMOC(r)
	if handled {
		return
	}

	var err = moc.Delete()
	if err != nil {
		logger.Errorf("Unable to delete MOC (%#v): %s", moc, err)
		r.Error(http.StatusInternalServerError, "Error trying to delete MOC - try again or contact support")
		return
	}

	r.Audit(models.AuditActionDeleteMoc, fmt.Sprintf("%#v", moc))
	http.SetCookie(w, &http.Cookie{Name: "Info", Value: "Deleted MOC", Path: "/"})
	http.Redirect(w, req, basePath, http.StatusFound)
}

func editHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var moc, handled = getMOC(r)
	if handled {
		return
	}

	r.Vars.Data["MOC"] = moc
	r.Vars.Title = "Editing MARC organization code"
	r.Render(formTmpl)
}

func getMOC(r *responder.Responder) (moc *models.MOC, handled bool) {
	var idStr = r.Request.FormValue("id")
	var id, _ = strconv.ParseInt(idStr, 10, 64)
	if id < 1 {
		logger.Warnf("Invalid MOC id for request %q (%s)", r.Request.URL.Path, idStr)
		r.Error(http.StatusBadRequest, "Invalid MOC id - try again or contact support")
		return nil, true
	}

	var err error
	moc, err = models.FindMOCByID(id)
	if err != nil {
		logger.Errorf("Unable to find MOC by id %d: %s", id, err)
		r.Error(http.StatusInternalServerError, "Unable to find MOC - try again or contact support")
		return nil, true
	}
	if moc == nil {
		r.Error(http.StatusNotFound, "Unable to find MOC - try again or contact support")
		return nil, true
	}

	return moc, false
}
