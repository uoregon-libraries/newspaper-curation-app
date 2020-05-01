package responder

import (
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/settings"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
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

func dict(values ...interface{}) (map[string]interface{}, error) {
	if len(values)%2 != 0 {
		return nil, errors.New("dict: values must be in pairs")
	}
	var dict = make(map[string]interface{}, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, errors.New("dict: keys must be strings")
		}
		dict[key] = values[i+1]
	}
	return dict, nil
}

// actionVerb fills in the blank when describing what happened, e.g., "kbates
// _rejected the issue metadata_ on Jan 1, 2002 at 3:45pm" or "jdepp _wrote a
// comment_"
func actionVerb(at string) string {
	switch models.ActionType(at) {
	case models.ActionTypeMetadataEntry:
		return "added metadata and pushed the issue to review"
	case models.ActionTypeMetadataApproval:
		return "approved the issue's metadata"
	case models.ActionTypeMetadataRejection:
		return "rejected the issue's metadata"
	case models.ActionTypeComment:
		return "wrote a comment"
	default:
		return string(at)
	}
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
		"dtstr":      func(t time.Time) string { return t.Format("on Jan 2, 2006 at 3:04pm") },
		"actionVerb": actionVerb,
		"nl2br": func(s string) template.HTML {
			var escaped = template.HTMLEscaper(s)
			var replaced = strings.Replace(escaped, "\n", "<br />", -1)
			return template.HTML(replaced)
		},
		"IIIFInfoURL": webutil.IIIFInfoURL,
		"raw":         func(s string) template.HTML { return template.HTML(s) },
		"debug":       func() bool { return settings.DEBUG },
		"dict":        dict,

		// We have functions for our privileges since they need to be "global" and
		// easily verified at template compile time
		"ListTitles":               func() *privilege.Privilege { return privilege.ListTitles },
		"ModifyTitles":             func() *privilege.Privilege { return privilege.ModifyTitles },
		"ManageMOCs":               func() *privilege.Privilege { return privilege.ManageMOCs },
		"ViewMetadataWorkflow":     func() *privilege.Privilege { return privilege.ViewMetadataWorkflow },
		"EnterIssueMetadata":       func() *privilege.Privilege { return privilege.EnterIssueMetadata },
		"ReviewIssueMetadata":      func() *privilege.Privilege { return privilege.ReviewIssueMetadata },
		"ListUsers":                func() *privilege.Privilege { return privilege.ListUsers },
		"ModifyUsers":              func() *privilege.Privilege { return privilege.ModifyUsers },
		"ViewUploadedIssues":       func() *privilege.Privilege { return privilege.ViewUploadedIssues },
		"ModifyUploadedIssues":     func() *privilege.Privilege { return privilege.ModifyUploadedIssues },
		"ViewTitleSFTPCredentials": func() *privilege.Privilege { return privilege.ViewTitleSFTPCredentials },
		"SearchIssues":             func() *privilege.Privilege { return privilege.SearchIssues },
		"ModifyValidatedLCCNs":     func() *privilege.Privilege { return privilege.ModifyValidatedLCCNs },
		"ModifyTitleSFTP":          func() *privilege.Privilege { return privilege.ModifyTitleSFTP },
		"ListAuditLogs":            func() *privilege.Privilege { return privilege.ListAuditLogs },
	}

	// Set up the layout and then our global templates
	Layout = tmpl.Root("layout", templatePath)
	Layout.Funcs(templateFunctions)
	Layout.MustReadPartials("layout.go.html")
	InsufficientPrivileges = Layout.MustBuild("insufficient-privileges.go.html")
	Empty = Layout.MustBuild("empty.go.html")
	Home = Layout.MustBuild("home.go.html")
}
