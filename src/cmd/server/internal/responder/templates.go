package responder

import (
	"fmt"
	"html/template"
	"path/filepath"
	"time"
	"web/webutil"
)

var templateFunctions template.FuncMap
var layout, InsufficientPrivileges, Empty *template.Template
var templatePath string

// HTMLComment forces an HTML comment into the source (since Go templates strip these)
func HTMLComment(s string) template.HTML {
	return template.HTML(fmt.Sprintf("<!-- %s -->", s))
}

// InitTemplates sets up pre-parsed template data - must be run after config has data
//
// TODO: Rewrite this; this is the wrong approach:
// - Different high-level areas are going to need their own function maps in
//   addition to a set of "core" functions
func InitTemplates(TemplatePath string) {
	templatePath = TemplatePath
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

	// Set up the layout and then our global templates
	layout = template.Must(t.ParseFiles(filepath.Join(templatePath, "layout.go.html")))
	InsufficientPrivileges = BuildTemplate("insufficient-privileges.go.html")
	Empty = BuildTemplate("empty.go.html")
}

// BuildTemplate returns a template compiled by combining our layout with the
// given path (relative to the template path)
func BuildTemplate(path string) *template.Template {
	var l2 = template.Must(layout.Clone())
	return template.Must(l2.ParseFiles(filepath.Join(templatePath, path)))
}
