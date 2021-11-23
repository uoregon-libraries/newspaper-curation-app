// Package responder contains all the general functionality necessary for
// responding to a given server request: template setup, user auth checks,
// rendering of pages to an http.ResponseWriter
package responder

import (
	"bytes"
	"encoding/base64"
	"html/template"
	"io"
	"net/http"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/version"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

// GenericVars holds anything specialized that doesn't make sense to have in PageVars
type GenericVars map[string]interface{}

// PageVars is the generic list of data all pages may need, and the catch-all
// "Data" map for specialized one-off data
type PageVars struct {
	Title   string
	Version string
	Alert   template.HTML
	Info    template.HTML
	User    *models.User
	Data    GenericVars
}

// Responder wraps common response logic
type Responder struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Vars    *PageVars
}

// Response generates a Responder with basic data all pages will need: request,
// response writer, and user
func Response(w http.ResponseWriter, req *http.Request) *Responder {
	var u = models.FindActiveUserWithLogin(GetUserLogin(w, req))
	u.IP = GetUserIP(req)
	return &Responder{Writer: w, Request: req, Vars: &PageVars{User: u, Data: make(GenericVars)}}
}

// injectDefaultTemplateVars sets up default variables used in multiple templates
func (r *Responder) injectDefaultTemplateVars() {
	r.Vars.Version = version.Version
	if r.Vars.Title == "" {
		r.Vars.Title = "Newspaper Curation App"
	}
}

// Render uses the responder's data to render the given template
func (r *Responder) Render(t *tmpl.Template) {
	r.injectDefaultTemplateVars()
	var cookie, err = r.Request.Cookie("Alert")
	if err == nil && cookie.Value != "" {
		r.Vars.Alert = template.HTML(cookie.Value)
		// TODO: This is such a horrible hack.  We need real session data management.
		if len(r.Vars.Alert) > 6 && r.Vars.Alert[0:6] == "base64" {
			var data, err = base64.StdEncoding.DecodeString(string(r.Vars.Alert[6:]))
			r.Vars.Alert = template.HTML(string(data))
			if err != nil {
				r.Vars.Alert = ""
			}
		}
		http.SetCookie(r.Writer, &http.Cookie{Name: "Alert", Value: "", Expires: time.Time{}, Path: "/"})
	}
	cookie, err = r.Request.Cookie("Info")
	if err == nil && cookie.Value != "" {
		r.Vars.Info = template.HTML(cookie.Value)
		http.SetCookie(r.Writer, &http.Cookie{Name: "Info", Value: "", Expires: time.Time{}, Path: "/"})
	}

	var buffer = new(bytes.Buffer)
	err = t.Execute(buffer, r.Vars)
	if err != nil {
		logger.Criticalf("Unable to render template %q: %s", t.Path, err)
		http.Error(r.Writer, "NCA has experienced an internal error while trying to render the page. Please contact the administrator for assistance.", http.StatusInternalServerError)
		return
	}
	_, err = io.Copy(r.Writer, buffer)
	if err != nil {
		logger.Errorf("Unable to copy template %q from buffer: %s", t.Name, err)
	}
}

// Audit stores an audit log in the database and logs to the command line if
// the database audit fails
func (r *Responder) Audit(action models.AuditAction, msg string) {
	var u = r.Vars.User
	var err = models.CreateAuditLog(u.IP, u.Login, action, msg)
	if err != nil {
		logger.Criticalf("Unable to write AuditLog{%s (%s), %q, %s}: %s", u.Login, u.IP, action, msg, err)
	}
}

// Error sets up the Alert var and sends the appropriate header to the browser.
// If msg is empty, the status text from the http package is used.
func (r *Responder) Error(status int, msg string) {
	r.Writer.WriteHeader(status)
	if msg == "" {
		msg = http.StatusText(status)
	}
	r.Vars.Alert = template.HTML(msg)
	r.Render(Empty)
}
