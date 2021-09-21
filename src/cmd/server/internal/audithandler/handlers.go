package audithandler

import (
	"fmt"
	"html/template"
	"net/http"
	"path"
	"time"

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

type form struct {
	PresetDate  string
	StartString string
	EndString   string
	Start       time.Time
	End         time.Time
	Valid bool
}

func parseCustomDate(f *form) error {
	var err error
	f.Start, err = time.Parse("2006-01-02", f.StartString)
	if err != nil {
		return fmt.Errorf("start date is missing or invalid")
	}

	f.End, err = time.Parse("2006-01-02", f.EndString)
	if err != nil {
		return fmt.Errorf("end date is missing or invalid")
	}

	if f.End.Before(f.Start) {
		return fmt.Errorf("start must come before end")
	}

		f.Valid = true
	return nil
}

// getForm stuffs the form data into our form structure for use in filtering
// and redisplaying the form
func getForm(r *responder.Responder) *form {
	var vfn = r.Request.FormValue
	var f = &form{
		PresetDate:  vfn("preset-date"),
		StartString: vfn("custom-date-start"),
		EndString:   vfn("custom-date-end"),
	}
	var now = time.Now()
	var min = time.Date(2010, 1, 1, 0, 0, 0, 0, time.Local)

	switch f.PresetDate {
	case "custom":
		var err = parseCustomDate(f)
		if err != nil {
			f.Start = min
			f.End = now
			r.Vars.Alert = template.HTML("Invalid date range: " + err.Error())
		}
	case "past12m":
		var y, m, d = now.Date()
		f.Start = time.Date(y-1, m, d, 0, 0, 0, 0, now.Location())
		f.End = now
	case "ytd":
		f.Start = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		f.End = now
	case "past30d":
		f.Start = now.Add(-time.Hour * 24 * 30)
		f.End = now
	case "today":
		var y, m, d = now.Date()
		f.Start = time.Date(y, m, d, 0, 0, 0, 0, now.Location())
		f.End = now
	default:
		f.Start = min
		f.End = now
		f.PresetDate = "all"
	}

	r.Vars.Data["Form"] = f
	return f
}

// listHandler shows the most recent list of audit logs
func listHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var f = getForm(r)

	// Set a useful title based on the request
	switch f.PresetDate {
	case "custom":
		if f.Valid {
		r.Vars.Title = fmt.Sprintf("Audit Logs: %s to %s", f.StartString, f.EndString)
		} else {
		r.Vars.Title = "Error Parsing Custom Date: Showing All Recent Logs"
		}
	case "past12m":
		r.Vars.Title = "Past 12 Months Audit Logs"
	case "ytd":
		r.Vars.Title = "This Year's Audit Logs"
	case "past30d":
		r.Vars.Title = "Past 30 Days Audit Logs"
	case "today":
		r.Vars.Title = "Today's Audit Logs"
	default:
		r.Vars.Title = "Recent Audit Logs"
	}

	// Get up to 100 audit logs
	var err error

	// if the requested range is "all", don't bother with the form's date values
	if f.PresetDate == "all" {
		r.Vars.Data["AuditLogs"], r.Vars.Data["AuditLogsCount"], err = models.FindRecentAuditLogs(100)
	} else {
		r.Vars.Data["AuditLogs"], r.Vars.Data["AuditLogsCount"], err = models.FindAuditLogsByDateRange(f.Start, f.End, 100)
	}
	if err != nil {
		logger.Errorf("Unable to load audit log list: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull audit logs - try again or contact support")
		return
	}

	r.Render(listTmpl)
}
