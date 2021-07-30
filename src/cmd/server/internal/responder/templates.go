package responder

import (
	"errors"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/settings"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
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
	return models.ActionType(at).Describe()
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
		"ErrorHTML":     errorHTML,
		"ErrorListHTML": errorListHTML,
		"IIIFInfoURL":   webutil.IIIFInfoURL,
		"raw":           func(s string) template.HTML { return template.HTML(s) },
		"debug":         func() bool { return settings.DEBUG },
		"dict":          dict,

		// This hack helps with dynamic heading - Go's templating system seems to
		// be confused when we have something like "<{{.Something}}>" - it decides
		// the brackets, despite not being in a variable, need to be escaped.
		"Open":  func(s string) template.HTML { return template.HTML("<" + s + ">") },
		"Close": func(s string) template.HTML { return template.HTML("</" + s + ">") },

		// We have functions for our privileges since they need to be "global" and
		// easily verified at template compile time
		"ListTitles":               func() *privilege.Privilege { return privilege.ListTitles },
		"ModifyTitles":             func() *privilege.Privilege { return privilege.ModifyTitles },
		"ManageMOCs":               func() *privilege.Privilege { return privilege.ManageMOCs },
		"ViewMetadataWorkflow":     func() *privilege.Privilege { return privilege.ViewMetadataWorkflow },
		"EnterIssueMetadata":       func() *privilege.Privilege { return privilege.EnterIssueMetadata },
		"ReviewIssueMetadata":      func() *privilege.Privilege { return privilege.ReviewIssueMetadata },
		"ReviewOwnMetadata":        func() *privilege.Privilege { return privilege.ReviewOwnMetadata },
		"ReviewUnfixableIssues":    func() *privilege.Privilege { return privilege.ReviewUnfixableIssues },
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

// errorHTML returns the error text - usually just err.Message(), but some
// errors (okay, just one for now) need more details, including HTML output
func errorHTML(err apperr.Error) template.HTML {
	var msg = template.HTMLEscapeString(err.Message())
	switch v := err.(type) {
	case *schema.DuplicateIssueError:
		if v.IsLive {
			// The location is the JSON we get from the web scanner, so we have to trim
			// ".json" off the end.  We could have the web view follow the JSON link to
			// get the unquestionably correct URL to the issue, but that would add tens
			// of thousands of unnecessary web hits.
			var nonJSONURL = v.Location[:len(v.Location)-5]
			msg += fmt.Sprintf(`: <a href="%s">%s</a>`, nonJSONURL, v.Name)
		}
	}

	return template.HTML(msg)
}

// errorListHTML returns HTML for errs joined together using errorHTML to let
// each error be displayed appropriately
func errorListHTML(errs apperr.List) template.HTML {
	var sList = make([]string, errs.Len())
	for i, err := range errs.All() {
		sList[i] = string(errorHTML(err))
	}
	return template.HTML(strings.Join(sList, "; "))
}
