// Package responder contains all the general functionality necessary for
// responding to a given server request: template setup, user auth checks,
// rendering of pages to an http.ResponseWriter
package responder

import (
	"log"
	"net/http"
	"user"
	"version"
	"web/tmpl"
	"web/webutil"
)

// GenericVars holds anything specialized that doesn't make sense to have in PageVars
type GenericVars map[string]interface{}

// PageVars is the generic list of data all pages may need, and the catch-all
// "Data" map for specialized one-off data
type PageVars struct {
	Title         string
	Version       string
	Webroot       string
	ParentWebroot string
	Alert         string
	User          *user.User
	Data          GenericVars
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
	var u = user.FindByLogin(GetUserLogin(w, req))
	return &Responder{Writer: w, Request: req, Vars: &PageVars{User: u, Data: make(GenericVars)}}
}

// injectDefaultTemplateVars sets up default variables used in multiple templates
func (r *Responder) injectDefaultTemplateVars() {
	r.Vars.Webroot = webutil.Webroot
	r.Vars.Version = version.Version
	if r.Vars.Title == "" {
		r.Vars.Title = "ODNP Admin"
	}
}

// Render uses the responder's data to render the given template
func (r *Responder) Render(t *tmpl.Template) {
	r.injectDefaultTemplateVars()

	var err = t.Execute(r.Writer, r.Vars)
	if err != nil {
		log.Printf("ERROR: Unable to render template %#v: %s", t.Name, err)
	}
}
