package audithandler

import (
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

var (
	basePath string
	conf     *config.Config

	// layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	layout *tmpl.TRoot

	// listTmpl is the template which shows audit logs
	listTmpl *tmpl.Template
)

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config) {
	conf = c
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(listHandler))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{"AuditHomeURL": func() string { return basePath }})
	layout.Path = path.Join(layout.Path, "audit")

	listTmpl = layout.MustBuild("list.go.html")
}

// canView is middleware to verify the user can view audit logs
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ListAuditLogs, h)
}

// listHandler shows the most recent list of audit logs
func listHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Recent Audit Logs"

	// Get most recent 100 audit logs
	var err error
	r.Vars.Data["AuditLogs"], r.Vars.Data["AuditLogsCount"], err = models.FindRecentAuditLogs(100)
	if err != nil {
		logger.Errorf("Unable to load audit log list: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull audit logs - try again or contact support")
		return
	}

	r.Render(listTmpl)
}
