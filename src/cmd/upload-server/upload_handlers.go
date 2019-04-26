package main

import (
	"net/http"

	"github.com/uoregon-libraries/gopkg/tmpl"
)

// Error is just a string that returns itself when its Error method is
// called so that const strings can implement the error interface
type Error string

func (e Error) Error() string {
	return string(e)
}

// errInvalidFormUID is used when a user is trying to load a form that
// doesn't exist (it may have expired or the server restarted or something)
const errInvalidFormUID = Error("invalid form uid")

// errUnownedForm occurs when a user has a form uid associated with a
// different user
const errUnownedForm = Error("user doesn't own requested form uid")

// Templates are global here because we need them accessible from multiple functions
var metadata *tmpl.Template
var upload *tmpl.Template

// uploadForm gets the form data from parseUploadForm.  On any errors, we
// automatically redirect the client and return ok==false so the caller knows
// to just exit, not handle anything further.
func (r *responder) uploadForm() (f *uploadForm, ok bool) {
	var err error
	f, err = r.getUploadForm()
	if err == nil {
		return f, true
	}

	switch err {
	case errInvalidFormUID:
		r.sess.SetAlertFlash("Unable to find session data - your form may have timed out")
		r.redirectSubpath("upload", http.StatusSeeOther)

	case errUnownedForm:
		r.redirectSubpath("upload", http.StatusSeeOther)

	case errInvalidDate:
		r.render(metadata, map[string]interface{}{
			"Form":  f,
			"Alert": "Invalid date.  If you are typing the date manually, please use MM-DD-YYYY or YYYY-MM-DD for formatting.",
		})

	default:
		r.server.logger.Errorf("Unknown error parsing form data: %#v", err)
		r.sess.SetAlertFlash("Unknown error parsing your form data.  Please reload and try again.")
		r.redirectSubpath("upload", http.StatusSeeOther)
	}

	return f, false
}

// getUploadForm retrieves the user's form from their session and populates it
// with their request data.  On backend or validation problems, an error is
// returned.
func (r *responder) getUploadForm() (*uploadForm, error) {
	var user = r.sess.GetString("user")

	// Retrieve form or register a new one
	var uid = r.req.FormValue("uid")

	// If the form is new, there can be no errors
	if uid == "" {
		return registerForm(user), nil
	}

	var f = findForm(uid)
	if f == nil {
		r.server.logger.Warnf("Session user %q trying to claim invalid form uid %q", user, uid)
		return nil, errInvalidFormUID
	}

	// Validate ownership
	if f != nil && f.Owner != user {
		r.server.logger.Errorf("Session user %q trying to claim form owned by %q", user, f.Owner)
		return nil, errUnownedForm
	}

	var err = f.parseRequest(r.req)
	return f, err
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
