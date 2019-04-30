package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

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

// getFormNonAJAX gets the form data from getUploadForm.  On any errors (not
// validation of data, but form errors), we automatically redirect the client
// and return ok==false so the caller knows to just exit, not handle anything
// further.
func (r *responder) getFormNonAJAX() (f *uploadForm, ok bool) {
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

	return f, nil
}

func (s *srv) uploadFormHandler() http.Handler {
	metadata = s.layout.MustBuild("upload-metadata.go.html")
	upload = s.layout.MustBuild("upload-files.go.html")

	return s.route(func(r *responder) {
		var form, ok = r.getFormNonAJAX()
		if !ok {
			return
		}

		var next = r.req.FormValue("nextstep")
		var data = map[string]interface{}{"Form": form}

		switch next {
		// If we're on the metadata step, we don't try to parse incoming fields or
		// validate the form
		case "", "metadata":
			r.render(metadata, data)

		case "upload":
			var err = form.parseMetadata(r.req)
			switch err {
			case nil:
				r.render(upload, data)
			case errInvalidDate:
				data["Alert"] = "Invalid date: make sure you use YYYY-MM-DD format"
				r.render(metadata, data)
			default:
				s.logger.Errorf("Unhandled form parse error: %s", err)
				data["Alert"] = "Invalid metadata"
				r.render(metadata, data)
			}

		default:
			s.logger.Warnf("Invalid next step: %q", next)
		}
	})
}

func (s *srv) uploadAJAXReceiver() http.Handler {
	return s.route(func(r *responder) {
		var uid = r.req.FormValue("uid")
		if uid == "" {
			s.logger.Errorf("AJAX request with no form uid!")
			r.ajaxError("upload error: no form", http.StatusBadRequest)
			return
		}

		var form, err = r.getUploadForm()
		if err != nil {
			s.logger.Errorf("Error processing form for AJAX request: %s", err)
			r.ajaxError("upload error: invalid form", http.StatusBadRequest)
			return
		}

		err = r.getAJAXUpload(form)
		if err != nil {
			s.logger.Errorf("Error reading file upload for AJAX request: %s", err)
			r.ajaxError("file upload error", http.StatusInternalServerError)
			return
		}

		r.w.Write([]byte("ok"))
	})
}

// getAJAXUpload pulls the AJAX upload and stores it into a temporary file.
// The path to the temp file is returned as well as any errors which occurred
// during the read/copy.
func (r *responder) getAJAXUpload(form *uploadForm) error {
	var file, header, err = r.req.FormFile("myfile")
	if err != nil {
		return err
	}
	r.server.logger.Infof("File upload: %q %d", header.Filename, header.Size)

	var out *os.File
	out, err = ioutil.TempFile(os.TempDir(), form.UID+"-")
	if err != nil {
		return fmt.Errorf("unable to create temp file for file upload: %s", err)
	}

	var n int64
	n, err = io.Copy(out, file)
	if n != header.Size {
		r.server.logger.Errorf("Wrote %d bytes to tempfile, but expected %d", n, header.Size)
		return fmt.Errorf("only wrote partial file")
	}
	if err != nil {
		r.server.logger.Errorf("Error writing to tempfile: %s", err)
		return fmt.Errorf("unable to write to tempfile")
	}

	err = out.Close()
	if err != nil {
		r.server.logger.Errorf("Error closing tempfile: %s", err)
		return fmt.Errorf("unable to close tempfile")
	}

	r.server.logger.Infof("Wrote %q to %q", header.Filename, out.Name())
	return nil
}
