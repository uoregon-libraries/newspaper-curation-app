package main

import (
	"net/http"
	"time"

	"github.com/uoregon-libraries/gopkg/tmpl"
)

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
