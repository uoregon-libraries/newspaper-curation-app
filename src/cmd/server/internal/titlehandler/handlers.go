package titlehandler

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/datasize"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/duration"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

var (
	basePath       string
	uploadMARCPath string
	conf           *config.Config

	// layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	layout *tmpl.TRoot

	// listTmpl is the template which shows all titles
	listTmpl *tmpl.Template

	// formTmpl is the form for adding or editing a title
	formTmpl *tmpl.Template

	// uploadMARCTmpl is the form for uploading a new MARC record to add to local
	// storage as well as sending on to ONI
	uploadMARCTmpl *tmpl.Template
)

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config) {
	conf = c
	basePath = baseWebPath
	uploadMARCPath = path.Join(basePath, "upload-marc")

	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(listHandler))
	s.Path("/new").Handler(canModify(newHandler))
	s.Path("/edit").Handler(canModify(editHandler))
	s.Path("/save").Methods("POST").Handler(canModify(saveHandler))
	s.Path("/validate").Methods("POST").Handler(canModify(validateHandler))
	s.Path("/upload-marc").Methods("GET").Handler(canModify(showMARCFormHandler))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{
		"TitlesHomeURL":       func() string { return basePath },
		"TitlesUploadMARCURL": func() string { return uploadMARCPath },
		"SFTPGoEnabled":       func() bool { return c.SFTPGoEnabled },
	})
	layout.Path = path.Join(layout.Path, "titles")

	listTmpl = layout.MustBuild("list.go.html")
	formTmpl = layout.MustBuild("form.go.html")
	uploadMARCTmpl = layout.MustBuild("upload-marc.go.html")
}

func getTitle(r *responder.Responder) (t *Title, handled bool) {
	var idStr = r.Request.FormValue("id")
	var id, _ = strconv.ParseInt(idStr, 10, 64)
	if id < 1 {
		logger.Warnf("Invalid title id for request %q (%s)", r.Request.URL.Path, idStr)
		r.Error(http.StatusBadRequest, "Invalid title id - try again or contact support")
		return nil, true
	}

	var dbt, err = models.FindTitleByID(id)
	if err != nil {
		logger.Errorf("Unable to find title by id %d: %s", id, err)
		r.Error(http.StatusInternalServerError, "Unable to find title - try again or contact support")
		return nil, true
	}
	if dbt == nil {
		r.Error(http.StatusNotFound, "Unable to find title - try again or contact support")
		return nil, true
	}

	var wrapped = WrapTitle(dbt)

	// If we've got a connection to SFTPGo, we have to read from there, too, not
	// just the database
	if conf.SFTPGoEnabled && dbt.SFTPConnected {
		var u, err = dbi.SFTP().GetUser(dbt.SFTPUser)
		if err != nil {
			logger.Errorf("Unable to look up title %q in SFTPGo: %s", dbt.SFTPUser, err)
			r.Error(http.StatusInternalServerError, "Unable to find title - try again or contact support")
			return nil, true
		}

		wrapped.SFTPQuota = datasize.Datasize(u.QuotaSize)
	}

	return wrapped, false
}

// listHandler spits out the list of titles
func listHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Titles"
	var dbTitles, err = models.Titles()
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
	r.Vars.Data["Title"] = WrapTitle(&models.Title{})
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

	t.EmbargoPeriod = form.Get("embargo_period")
	var embargoPeriod duration.Duration
	embargoPeriod, err = duration.Parse(t.EmbargoPeriod)
	if err != nil {
		vErrors = append(vErrors, fmt.Sprintf("Embargo period is invalid: %s", err))
	} else {
		t.EmbargoPeriod = embargoPeriod.String()
	}

	if conf.SFTPGoEnabled {
		var newUser = form.Get("sftpuser")
		if newUser != "" && !t.SFTPConnected {
			t.SFTPUser = newUser
		}
		t.SFTPPass = form.Get("sftppass")

		var raw = form.Get("sftpquota")
		if raw == "" {
			vErrors = append(vErrors, "SFTP quota cannot be blank")
		}
		var quota, err = datasize.New(raw)
		if err == nil {
			t.SFTPQuota = quota
		} else {
			vErrors = append(vErrors, fmt.Sprintf("Invalid SFTP quota %q: %s", raw, err))
		}
	}

	if !t.ValidLCCN || r.Vars.User.PermittedTo(privilege.ModifyValidatedLCCNs) {
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

	var allTitles []*models.Title
	allTitles, err = models.Titles()
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
		if t.SFTPUser != "" && t.SFTPUser == t2.SFTPUser {
			vErrors = append(vErrors, fmt.Sprintf("SFTP Username %s is already in use", t.SFTPUser))
		}
	}

	return vErrors, false
}

// saveHandler inserts or updates a title in the db, removing sensitive form
// data for users who can't edit it (sftp credentials)
func saveHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var t = WrapTitle(&models.Title{})
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

	// Title saving is complex because of SFTPGo integration, so it's tucked away
	var msg, err = saveTitle(t)
	if err != nil {
		logger.Errorf("Unable to save title %q: %s", t.Name, err)
		r.Vars.Data["Title"] = t
		r.Vars.Title = "Error saving title " + t.Name
		r.Vars.Alert = template.HTML(msg)
		r.Render(formTmpl)
		return
	}

	// We only check on MARC XML if we were able to save successfully to the
	// database; this data is useful, but not critical to NCA's operations, so we
	// run it in the background and let it do its thing when it can.  This should
	// probably be a new job or something, though.
	if !t.ValidLCCN {
		go pullMARCForTitle(t)
	}

	r.Audit(models.AuditActionSaveTitle, fmt.Sprintf("%#v", r.Request.Form))

	http.SetCookie(w, &http.Cookie{
		Name:  "Info",
		Value: "base64" + base64.StdEncoding.EncodeToString([]byte(msg)),
		Path:  "/",
	})
	http.Redirect(w, r.Request, basePath, http.StatusFound)
}

func saveTitle(t *Title) (msg string, err error) {
	// If there's no SFTPGo connection, we just save and return
	if !conf.SFTPGoEnabled {
		return "Title saved", t.Save()
	}

	// If the title isn't connected, but username is blank, we also return
	if !t.SFTPConnected && t.SFTPUser == "" {
		return "Title saved", t.Save()
	}

	// We connect to SFTPGo, so we need a transaction
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()

	// We'll need to save the title, and set its connection flag to true
	var wasConnected = t.SFTPConnected
	t.SFTPConnected = true
	err = t.SaveOp(op)
	var sftpMessage string
	if err != nil {
		return "database write failure", err
	}

	sftpMessage, err = integrateSFTP(t, wasConnected)
	if err != nil {
		// rollback and set the in-memory title's sftp connection to its prior value
		op.Rollback()
		t.SFTPConnected = wasConnected
		return "couldn't integrate title into SFTP server", fmt.Errorf("Error in SFTPGo integration for title %q (SFTPUser %q): %w", t.Name, t.SFTPUser, err)
	}

	op.EndTransaction()
	return "Title saved.  SFTP Integration successful: " + sftpMessage, op.Err()
}

// integrateSFTP attempts to create or update a user in SFTPGo
func integrateSFTP(t *Title, connected bool) (msg string, err error) {
	if !conf.SFTPGoEnabled {
		return "", nil
	}

	// If the title already has an SFTP connection, we perform an update
	if connected {
		err = dbi.SFTP().UpdateUser(t.SFTPUser, t.SFTPPass, int64(t.SFTPQuota))
		if err != nil {
			return fmt.Sprintf("Error updating SFTP password for user %q: try again or contact support", t.SFTPUser), err
		}
		return "update complete", nil
	}

	var pass string
	pass, err = dbi.SFTP().CreateUser(t.SFTPUser, t.SFTPPass, int64(t.SFTPQuota), t.Name+" / "+t.LCCN)
	if err != nil {
		return fmt.Sprintf("Error provisioning the SFTP user %q: try again or contact support", t.SFTPUser), err
	}
	return fmt.Sprintf("SFTP credentials: Username %q; Password %q", t.SFTPUser, pass), nil
}

func validateHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var t, handled = getTitle(r)
	if handled {
		return
	}

	// When validation is explicitly requested, the user waits for a response
	pullMARCForTitle(t)
	r.Audit(models.AuditActionValidateTitle, fmt.Sprintf("%q %q", t.MARCTitle, t.MARCLocation))

	var alertLevel = "Info"
	var response = "Validated LCCN"
	if !t.ValidLCCN {
		alertLevel = "Alert"
		response = "LCCN was not able to be validated at this time - Chronicling America may be down or the LCCN may not be in their database"
	}
	http.SetCookie(w, &http.Cookie{Name: alertLevel, Value: response, Path: "/"})
	http.Redirect(w, r.Request, basePath, http.StatusFound)
}

func showMARCFormHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Upload MARC XML"
	r.Render(uploadMARCTmpl)
}
