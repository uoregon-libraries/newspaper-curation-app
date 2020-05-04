package responder

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/settings"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models/user"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/webutil"
)

var (
	// Layout holds the base site layout template.  Handlers should clone and use
	// this for parsing their specific page templates
	Layout *tmpl.TRoot

	// InsufficientPrivileges is a simple page to declare to a user they are not
	// allowed to visit a certain page or perform a certain action
	InsufficientPrivileges *tmpl.Template

	// Home (for now) is a very simple static welcome page
	Home *tmpl.Template

	// Empty holds a simple blank page for rendering the header/footer and often
	// a simple alert-style message
	Empty *tmpl.Template
)

// HTMLComment forces an HTML comment into the source (since Go templates strip these)
func HTMLComment(s string) template.HTML {
	return template.HTML(fmt.Sprintf("<!-- %s -->", s))
}

// InitRootTemplate sets up pre-parsed template data in Root
func InitRootTemplate(templatePath string) {
	var templateFunctions = tmpl.FuncMap{
		"IncludeCSS": webutil.IncludeCSS,
		"RawCSS":     webutil.RawCSS,
		"IncludeJS":  webutil.IncludeJS,
		"RawJS":      webutil.RawJS,
		"ImageURL":   webutil.ImageURL,
		"HomePath":   webutil.HomePath,
		"FullPath":   webutil.FullPath,
		"ProdURL":    func() string { return webutil.ProductionURL },
		"Comment":    HTMLComment,
		"TimeString": func(t time.Time) string { return t.Format("2006-01-02 15:04") },
		"nl2br": func(s string) template.HTML {
			var escaped = template.HTMLEscaper(s)
			var replaced = strings.Replace(escaped, "\n", "<br />", -1)
			return template.HTML(replaced)
		},
		"IIIFInfoURL": webutil.IIIFInfoURL,
		"raw":         func(s string) template.HTML { return template.HTML(s) },
		"debug":       func() bool { return settings.DEBUG },

		// We have functions for our privileges since they need to be "global" and
		// easily verified at template compile time
		"ListTitles":               func() *user.Privilege { return user.ListTitles },
		"ModifyTitles":             func() *user.Privilege { return user.ModifyTitles },
		"ManageMOCs":               func() *user.Privilege { return user.ManageMOCs },
		"ViewMetadataWorkflow":     func() *user.Privilege { return user.ViewMetadataWorkflow },
		"EnterIssueMetadata":       func() *user.Privilege { return user.EnterIssueMetadata },
		"ReviewIssueMetadata":      func() *user.Privilege { return user.ReviewIssueMetadata },
		"ListUsers":                func() *user.Privilege { return user.ListUsers },
		"ModifyUsers":              func() *user.Privilege { return user.ModifyUsers },
		"ViewUploadedIssues":       func() *user.Privilege { return user.ViewUploadedIssues },
		"ModifyUploadedIssues":     func() *user.Privilege { return user.ModifyUploadedIssues },
		"ViewTitleSFTPCredentials": func() *user.Privilege { return user.ViewTitleSFTPCredentials },
		"SearchIssues":             func() *user.Privilege { return user.SearchIssues },
		"ModifyValidatedLCCNs":     func() *user.Privilege { return user.ModifyValidatedLCCNs },
		"ModifyTitleSFTP":          func() *user.Privilege { return user.ModifyTitleSFTP },
		"ListAuditLogs":            func() *user.Privilege { return user.ListAuditLogs },
	}

	// Set up the layout and then our global templates
	Layout = tmpl.Root("layout", templatePath)
	Layout.Funcs(templateFunctions)
	Layout.MustReadPartials("layout.go.html")
	InsufficientPrivileges = Layout.MustBuild("insufficient-privileges.go.html")
	Empty = Layout.MustBuild("empty.go.html")
	Home = Layout.MustBuild("home.go.html")
}
