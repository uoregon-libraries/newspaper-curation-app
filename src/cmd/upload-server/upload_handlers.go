package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/tmpl"
)

// uploadForm holds onto all validated data the form has already handled.  Form
// parameters will overwrite anything previously validated to account for going
// back/forward on the form.
type uploadForm struct {
	sync.Mutex
	Owner        string
	Date         time.Time
	UID          string
	lastAccessed time.Time
}

func (f *uploadForm) access() {
	f.Lock()
	f.lastAccessed = time.Now()
	f.Unlock()
}

func (f *uploadForm) String() string {
	return fmt.Sprintf(`&uploadForm{"Owner": %q, "Date": %q, "UID": %q, "lastAccessed": %q}`,
		f.Owner, f.Date.Format("2006-01-02"), f.UID, f.lastAccessed.String())
}

var forml sync.Mutex
var forms = make(map[string]*uploadForm)

func registerForm(owner string) *uploadForm {
	var f = &uploadForm{Owner: owner, lastAccessed: time.Now(), UID: genid()}
	forml.Lock()
	forms[f.UID] = f
	forml.Unlock()

	logger.Infof("Registering new form %s", f)
	return f
}

func findForm(uid string) *uploadForm {
	forml.Lock()
	var f = forms[uid]
	forml.Unlock()

	if f != nil {
		f.access()
	}
	return f
}

func cleanForms() {
	for {
		purgeOldForms()
		time.Sleep(time.Hour * 6)
	}
}

func purgeOldForms() {
	var expired = time.Now().Add(-48 * time.Hour)
	logger.Debugf("Purging old forms from before %s", expired)
	forml.Lock()
	var keysToPurge []string
	for key, f := range forms {
		logger.Debugf("Looking at form %s", f)
		if f.lastAccessed.Before(expired) {
			logger.Debugf("Will purge")
			keysToPurge = append(keysToPurge, key)
		} else {
			logger.Debugf("Will *not* purge")
		}
	}

	for _, key := range keysToPurge {
		delete(forms, key)
	}
	forml.Unlock()
}

// Templates are global here because we need them accessible from multiple functions
var metadata *tmpl.Template
var upload *tmpl.Template

// uploadForm gets the form from the session and overwrites it with any request
// data.  If any data is invalid, we automatically redirect the client and
// return ok==false so the caller knows to just exit, not handle anything
// further.
func (r *responder) uploadForm() (f *uploadForm, ok bool) {
	var user = r.sess.GetString("user")

	// Retrieve form or register a new one
	var uid = r.req.FormValue("uid")
	if uid != "" {
		f = findForm(uid)
		if f == nil {
			r.server.logger.Errorf("Session user %q trying to claim invalid form uid %q", user, uid)
			r.sess.SetAlertFlash("Unable to find session data - your form may have timed out")
			r.redirectSubpath("upload", http.StatusSeeOther)
			return nil, false
		}

		// Validate ownership
		if f != nil && f.Owner != user {
			r.server.logger.Errorf("Session user %q trying to claim form owned by %q", user, f.Owner)
			r.redirectSubpath("upload", http.StatusSeeOther)
			return nil, false
		}
	} else {
		f = registerForm(r.sess.GetString("user"))
	}

	// Apply request form values if any are present
	var rawDate = r.req.FormValue("date")
	var date, err = time.Parse("2006-01-02", rawDate)
	if err != nil {
		date, err = time.Parse("01-02-2006", rawDate)
		if err != nil {
			r.render(metadata, map[string]interface{}{
				"Form":  f,
				"Alert": "Invalid date.  If you are typing the date manually, please use MM-DD-YYYY or YYYY-MM-DD for formatting.",
			})
			return f, false
		}
	}

	// Lock the form and assign the data
	f.Lock()
	f.Date = date
	f.Unlock()

	return f, true
}

func (s *srv) uploadFormHandler() http.Handler {
	metadata = s.layout.MustBuild("upload-metadata.go.html")
	upload = s.layout.MustBuild("upload-files.go.html")

	return s.respond(func(r *responder) {
		// We always process all form data since any form step can go backwards
		var form, ok = r.uploadForm()
		if !ok {
			return
		}

		switch r.req.FormValue("nextstep") {
		case "", "metadata":
			r.render(metadata, map[string]interface{}{"Form": form})
		case "upload":
			r.render(upload, map[string]interface{}{"Form": form})
		case "confirm":
		}
	})
}
