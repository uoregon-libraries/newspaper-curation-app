package responder

import (
	"fmt"
	"html/template"
	"path/filepath"
	"time"
	"web/tmpl"
	"web/webutil"
)

var Layout *tmpl.TRoot
var InsufficientPrivileges, Empty *tmpl.Template

// HTMLComment forces an HTML comment into the source (since Go templates strip these)
func HTMLComment(s string) template.HTML {
	return template.HTML(fmt.Sprintf("<!-- %s -->", s))
}

// InitRootTemplate sets up pre-parsed template data in Root
func InitRootTemplate(templatePath string) {
	var templateFunctions = template.FuncMap{
		"IncludeCSS": webutil.IncludeCSS,
		"RawCSS":     webutil.RawCSS,
		"IncludeJS":  webutil.IncludeJS,
		"RawJS":      webutil.RawJS,
		"ImageURL":   webutil.ImageURL,
		"ParentURL":  webutil.ParentURL,
		"Comment":    HTMLComment,
		"TimeString": func(t time.Time) string { return t.Format("2006-01-02 15:04") },
	}

	// Set up the layout and then our global templates
	Layout = tmpl.Root("layout", templatePath, templateFunctions)
	Layout.MustReadPartials("layout.go.html")
	template.Must(Layout.ParseFiles(filepath.Join(templatePath, "layout.go.html")))
	InsufficientPrivileges = Layout.MustBuild("insufficient-privileges.go.html")
	Empty = Layout.MustBuild("empty.go.html")
}
