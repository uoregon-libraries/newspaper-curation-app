package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
)

// PageVars is the generic list of data all pages may need
type PageVars struct {
	Title   string
	Version string
	Webroot string
	User    *User
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
func initTemplates(Webroot, TemplatePath string) {
	templateFunctions = template.FuncMap{
		"Comment": HTMLComment,
	}
	var t = template.New("root").Funcs(templateFunctions)
	templates = template.Must(t.ParseGlob(TemplatePath + "/*.go.html"))
}

// injectDefaultTemplateVars sets up default variables used in multiple templates
func (r *Responder) injectDefaultTemplateVars() {
	r.Vars.Webroot = Webroot
	r.Vars.Version = version
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
