package audithandler

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
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
	s.Path("/csv").Handler(canView(csvHandler))

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
	Valid       bool
}

// QueryString encodes the form values for reuse in an href
func (f *form) QueryString() template.URL {
	var v = url.Values{}
	v.Set("preset-date", f.PresetDate)
	if f.PresetDate == "custom" {
		v.Set("custom-date-start", f.StartString)
		v.Set("custom-date-end", f.EndString)
	}

	logger.Infof(v.Encode())
	return template.URL(v.Encode())
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

	// Make sure the custom dates are helpful if the user wants to switch from a
	// preset to custom
	if f.StartString == "" {
		f.StartString = f.Start.Format("2006-01-02")
	}
	if f.EndString == "" {
		f.EndString = f.End.Format("2006-01-02")
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

	// Get up to 100 audit logs. If the requested range is "all", don't bother
	// with the form's date values.
	var finder = models.AuditLogs().Limit(100)
	var err error
	if f.PresetDate != "all" {
		finder = finder.Between(f.Start, f.End)
	}
	r.Vars.Data["AuditLogs"], r.Vars.Data["AuditLogsCount"], err = finder.All()
	if err != nil {
		logger.Errorf("Unable to load audit log list: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to pull audit logs - try again or contact support")
		return
	}

	r.Render(listTmpl)
}

// csvHandler creates and streams a CSV of all audit logs matching the query
func csvHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var f = getForm(r)

	// Pull all logs matching the request. If the requested range is "all", don't
	// bother with the form's date values.
	var err error
	var logs []*models.AuditLog
	if f.PresetDate == "all" {
		logs, _, err = models.AuditLogs().All()
	} else {
		logs, _, err = models.AuditLogs().Between(f.Start, f.End).All()
	}
	if err != nil {
		logger.Errorf("Unable to load audit log list: %s", err)
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
		cw.Write([]string{l.When.Format("2006-01-02 15:04"), l.User, l.IP, l.Action, l.Message})
	}
}
