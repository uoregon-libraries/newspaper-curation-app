package audithandler

import (
	"fmt"
	"html/template"
	"net/url"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

type form struct {
	PresetDate  string
	StartString string
	EndString   string
	Username    string
	ActionTypes string
	Start       time.Time
	End         time.Time
	Invalid     bool
}

var actionLookup = map[string][]models.AuditAction{
	"Uploads":        {models.AuditActionQueue},
	"Titles":         {models.AuditActionSaveTitle, models.AuditActionValidateTitle},
	"MARC Org Codes": {models.AuditActionCreateMoc, models.AuditActionUpdateMoc, models.AuditActionDeleteMoc},
	"Users":          {models.AuditActionSaveUser, models.AuditActionDeactivateUser},
	"Issue Workflow": {
		models.AuditActionClaim,
		models.AuditActionUnclaim,
		models.AuditActionApproveMetadata,
		models.AuditActionRejectMetadata,
		models.AuditActionReportError,
		models.AuditActionUndoErrorIssue,
		models.AuditActionRemoveErrorIssue,
		models.AuditActionQueueForReview,
		models.AuditActionSaveDraft,
		models.AuditActionSaveQueue,
	},
}

// getForm stuffs the form data into our form structure for use in filtering
// and redisplaying the form
func getForm(r *responder.Responder) *form {
	var vfn = r.Request.FormValue
	var f = &form{
		PresetDate:  vfn("preset-date"),
		StartString: vfn("custom-date-start"),
		EndString:   vfn("custom-date-end"),
		Username:    vfn("user"),
		ActionTypes: vfn("action-types"),
	}
	var now = time.Now()
	var minTime = time.Date(2010, 1, 1, 0, 0, 0, 0, time.Local)

	switch f.PresetDate {
	case "custom":
		var err = f.parseCustomDate()
		if err != nil {
			f.Start = minTime
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
		f.Start = minTime
		f.End = now
		f.PresetDate = "all"
	}

	// Make sure the custom dates are helpful if the user wants to switch from a
	// preset to custom
	if f.PresetDate != "custom" {
		f.StartString = f.Start.Format("2006-01-02")
		f.EndString = f.End.Format("2006-01-02")
	}

	r.Vars.Data["Form"] = f
	return f
}

// QueryString encodes the form values for reuse in an href
func (f *form) QueryString() template.URL {
	var v = url.Values{}
	v.Set("preset-date", f.PresetDate)
	if f.PresetDate == "custom" {
		v.Set("custom-date-start", f.StartString)
		v.Set("custom-date-end", f.EndString)
	}
	v.Set("user", f.Username)
	v.Set("action-types", f.ActionTypes)

	logger.Infof(v.Encode())
	return template.URL(v.Encode())
}

// title returns a useful title / caption based on the request
func (f *form) title() string {
	var logText = "Audit Logs"
	if f.ActionTypes != "" {
		logText = f.ActionTypes + " Audit Logs"
	}

	var title = "Recent {{logs}}"
	switch f.PresetDate {
	case "custom":
		title = fmt.Sprintf("{{logs}}: %s to %s", f.StartString, f.EndString)
		if f.Invalid {
			title = "Error Parsing Custom Date: Showing Recent {{logs}}"
		}
	case "past12m":
		title = "Past 12 Months {{logs}}"
	case "ytd":
		title = "This Year's {{logs}}"
	case "past30d":
		title = "Past 30 Days {{logs}}"
	case "today":
		title = "Today's {{logs}}"
	}

	if f.Username != "" {
		title += " for " + f.Username
	}
	title = strings.Replace(title, "{{logs}}", logText, 1)

	return title
}

func (f *form) logs(limit int) ([]*models.AuditLog, uint64, error) {
	var finder = models.AuditLogs()
	if f.PresetDate != "all" {
		finder.Between(f.Start, f.End)
	}
	if f.Username != "" {
		finder.ForUser(f.Username)
	}
	if f.ActionTypes != "" {
		finder.ForActions(actionLookup[f.ActionTypes]...)
	}
	if limit > 1 {
		finder.Limit(limit)
	}

	var logs, count, err = finder.Fetch()
	if err != nil {
		logger.Errorf("Unable to load audit log list: %s", err)
	}
	return logs, count, err
}

func (f *form) parseCustomDate() error {
	f.Invalid = true
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

	f.Invalid = false
	return nil
}
