package titlehandler

import (
	"cmd/server/internal/responder"
	"config"
	"db"
	"db/user"
	"fmt"
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
	s.Path("/save").Methods("POST").Handler(canModify(saveHandler))
	s.Path("/validate").Methods("POST").Handler(canModify(validateHandler))

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

// setTitleData grabs all the form values and applies them to the title.  Only
// permitted fields are used, based on the user's privileges and the state of
// the title.
//
// Any fields which fail validation will have an explanation added to the
// validationErrors list, which should be displayed to the user, and the title
// should not be saved.
//
// If a "real" error occurs, it is logged, a message is sent to the client's
// browser, and `handled` is true.
//
// TODO: This is a giant mix of validation, error checking, and business logic.
// Ideally we would split this all up into multiple places.  That would mean
// adding some annoying indirection, though: having to check if the DB
// validations are good separately from the client data entry separately from
// dealing with business logic which is intrinsically tied to the data
// entry....  Let's think on this some.
func setTitleData(r *responder.Responder, t *Title) (vErrors []string, handled bool) {
	var err = r.Request.ParseForm()
	if err != nil {
		logger.Errorf("Unable to parse title form: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to save title data - try again or contact support")
		return nil, true
	}

	var form = r.Request.Form
	t.Name = form.Get("name")
	t.Rights = form.Get("rights")

	switch form.Get("embargoed") {
	case "0":
		t.Embargoed = false
	case "1":
		t.Embargoed = true
	default:
		vErrors = append(vErrors, "You must declare whether this title embargoes its issues")
	}

	if r.Vars.User.PermittedTo(user.ModifyTitleSFTP) {
		t.SFTPDir = form.Get("sftpdir")
		t.SFTPUser = form.Get("sftpuser")
		t.SFTPPass = form.Get("sftppass")
	}

	if !t.ValidLCCN || r.Vars.User.PermittedTo(user.ModifyValidatedLCCNs) {
		var newLCCN = form.Get("lccn")
		if newLCCN != t.LCCN {
			t.LCCN = newLCCN
			t.ValidLCCN = false
		}
	}

	if t.Name == "" {
		vErrors = append(vErrors, "Name cannot be blank")
	}
	if t.LCCN == "" {
		vErrors = append(vErrors, "LCCN cannot be blank")
	}

	var allTitles []*db.Title
	allTitles, err = db.Titles()
	if err != nil {
		logger.Errorf("Unable to check database for title dupes: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to save title data - try again or contact support")
		return nil, true
	}

	for _, t2 := range allTitles {
		if t.ID == t2.ID {
			continue
		}
		if t.LCCN == t2.LCCN {
			vErrors = append(vErrors, fmt.Sprintf("LCCN %q is already in use", t.LCCN))
		}
		if t.SFTPDir != "" && t.SFTPDir == t2.SFTPDir {
			vErrors = append(vErrors, fmt.Sprintf("SFTPDir %q is already in use", t.SFTPDir))
		}
	}

	return vErrors, false
}

// saveHandler inserts or updates a title in the db, removing sensitive form
// data for users who can't edit it (sftp credentials)
func saveHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var t = WrapTitle(&db.Title{})
	var handled bool

	if r.Request.FormValue("id") != "" {
		t, handled = getTitle(r)
		if handled {
			return
		}
	}

	var validationErrors []string
	validationErrors, handled = setTitleData(r, t)
	if handled {
		return
	}
	if len(validationErrors) > 0 {
		r.Vars.Data["ValidationErrors"] = validationErrors
		r.Vars.Data["Title"] = t
		r.Vars.Title = "Editing " + t.Name
		r.Render(formTmpl)
		return
	}

	var err = t.Save()
	if err != nil {
		logger.Errorf("Unable to save title %q: %s", t.Name, err)
		r.Error(http.StatusInternalServerError, "Error trying to save title - try again or contact support")
		return
	}

	// We only check on MARC XML if we were able to save successfully to the
	// database; this data is useful, but not critical to NCA's operations, so we
	// run it in the background and let it do its thing when it can.  This should
	// probably be a new job or something, though.
	if !t.ValidLCCN {
		go pullMARCForTitle(t)
	}

	r.Audit("save-title", fmt.Sprintf("%#v", r.Request.Form))
	http.SetCookie(w, &http.Cookie{Name: "Info", Value: "Title saved", Path: "/"})
	http.Redirect(w, r.Request, basePath, http.StatusFound)
}

func validateHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var t, handled = getTitle(r)
	if handled {
		return
	}

	// When validation is explicitly requested, the user waits for a response
	pullMARCForTitle(t)
	r.Audit("validate-title", fmt.Sprintf("%q %q", t.MARCTitle, t.MARCLocation))

	var alertLevel = "Info"
	var response = "Validated LCCN"
	if !t.ValidLCCN {
		alertLevel = "Alert"
		response = "LCCN was not able to be validated at this time - Chronicling America may be down or the LCCN may not be in their database"
	}
	http.SetCookie(w, &http.Cookie{Name: alertLevel, Value: response, Path: "/"})
	http.Redirect(w, r.Request, basePath, http.StatusFound)
}
