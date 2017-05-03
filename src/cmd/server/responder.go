package main

import (
	"cmd/server/internal/presenter"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
	"user"
	"version"
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
	Titles        []*presenter.Title
	Data          GenericVars
}

// Responder wraps common response logic
type Responder struct {
	Writer  http.ResponseWriter
	Request *http.Request
	Vars    *PageVars
}

var templateFunctions template.FuncMap
var templates *template.Template

// HTMLComment forces an HTML comment into the source (since Go templates strip these)
func HTMLComment(s string) template.HTML {
	return template.HTML(fmt.Sprintf("<!-- %s -->", s))
}

// initTemplates sets up pre-parsed template data - must be run after config has data
//
// TODO: Rewrite this; this is the wrong approach:
// - There should be multiple templates instead of one that gloms together all files
// - Each template should use a layout rather than the inclusion of "header" and "footer"
// - Different high-level areas are going to need their own function maps in
//   addition to a set of "core" functions
func initTemplates(TemplatePath string) {
	templateFunctions = template.FuncMap{
		"Permitted":  func(user interface{}, action string) bool { return false },
		"IncludeCSS": webutil.IncludeCSS,
		"RawCSS":     webutil.RawCSS,
		"IncludeJS":  webutil.IncludeJS,
		"RawJS":      webutil.RawJS,
		"ImageURL":   webutil.ImageURL,
		"ParentURL":  webutil.ParentURL,
		"Comment":    HTMLComment,
		"TimeString": func(t time.Time) string { return t.Format("2006-01-02 15:04") },
	}
	var t = template.New("root").Funcs(templateFunctions)
	templates = template.Must(t.ParseGlob(TemplatePath + "/*.go.html"))
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
func (r *Responder) Render(name string) {
	var t = templates.Lookup(name + ".go.html")
	if t == nil {
		log.Printf("ERROR: Template %s requested but does not exist!", name)
		http.Error(r.Writer, "Error rendering the page", http.StatusInternalServerError)
		return
	}

	r.injectDefaultTemplateVars()

	var err = t.Execute(r.Writer, r.Vars)
	if err != nil {
		log.Printf("ERROR: Unable to render template %s: %s", name, err)
	}
}
