package audithandler

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
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

// canView is middleware to verify the user can view audit logs
func canView(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.ListAuditLogs, h)
}

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config) {
	conf = c
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(listHandler))
	s.Path("/csv").Handler(canView(csvHandler))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{"AuditHomeURL": func() string { return basePath }})
	layout.Path = path.Join(layout.Path, "audit")

	listTmpl = layout.MustBuild("list.go.html")
}

// listHandler shows the most recent list of audit logs
func listHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var f = getForm(r)
	r.Vars.Title = f.title()

	var err error
	r.Vars.Data["AuditLogs"], r.Vars.Data["AuditLogsCount"], err = f.logs(100)
	if err != nil {
		r.Error(http.StatusInternalServerError, "Error trying to pull audit logs - try again or contact support")
		return
	}

	r.Render(listTmpl)
}

// csvHandler creates and streams a CSV of all audit logs matching the query
func csvHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var f = getForm(r)
	var logs, _, err = f.logs(-1)
	if err != nil {
		r.Error(http.StatusInternalServerError, "Error trying to generate audit log CSV - try again or contact support")
		return
	}

	// Set up headers so the browser knows to download it
	var fname = fmt.Sprintf("logs-%s-%s.csv", f.Start.Format("20060102"), f.End.Format("20060102"))
	w.Header().Add("Content-Type", "text/csv")
	w.Header().Add("Content-Disposition", `attachment; filename="`+fname+`"`)
	var cw = csv.NewWriter(w)

	cw.Write([]string{"When", "User", "IP Address", "Action", "Raw Message"})
	for _, l := range logs {
		cw.Write([]string{l.When.Format("2006-01-02 15:04"), l.User, l.IP, l.Action, strings.Replace(l.Message, "\n", "\\n", -1)})
	}
	cw.Flush()
}
