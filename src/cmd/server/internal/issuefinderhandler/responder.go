package issuefinderhandler

import (
	"cmd/server/internal/responder"
	"db"
	"html/template"
	"net/http"
	"path"
	"schema"
	"strconv"
	"web/tmpl"

	"github.com/uoregon-libraries/gopkg/logger"
)

type respError struct {
	status int
	msg    string
}

// resp wraps responder.Responder to add in some data that is useful to
// auto-load in all search responses
type resp struct {
	*responder.Responder
	err    *respError
	Titles schema.TitleList
	Issues schema.IssueList
	LCCN   string
	Year   int
	Month  int
	Day    int
}

// getResponder sets up a resp with sane defaults
func getResponder(w http.ResponseWriter, req *http.Request) *resp {
	var r = &resp{Responder: responder.Response(w, req)}
	r.loadTitles()
	r.setupResults()
	return r
}

// loadTitles grabs all known titles from the database, converts to
// schema.Title instances, and stuffs them into a "Titles" variable
func (r *resp) loadTitles() {
	var titles = make(schema.TitleList, 0)
	var dbTitles, err = db.AllTitles()
	if err != nil {
		logger.Errorf("Unable to look up titles from database: %s", err)
		r.err = &respError{
			status: http.StatusInternalServerError,
			msg:    "Error trying to look up titles.  Try again or contact support",
		}
		return
	}

	for _, t := range dbTitles {
		titles = append(titles, t.SchemaTitle())
	}
	titles.SortByName()

	r.Titles = titles
}

func (r *resp) setupResults() {
	var err error
	var vfn = r.Request.FormValue
	var lccn = vfn("lccn")
	if lccn == "" {
		return
	}

	var syear, smonth, sday string
	syear = vfn("year")
	smonth = vfn("month")
	sday = vfn("day")

	var year, month, day int
	if syear != "" {
		year, err = strconv.Atoi(syear)
	}
	if smonth != "" && err == nil {
		month, err = strconv.Atoi(smonth)
	}
	if sday != "" && err == nil {
		day, err = strconv.Atoi(sday)
	}

	if err != nil {
		r.Vars.Alert = template.HTML("Invalid search: year, month, and day must be numeric or blank")
		return
	}
	if month > 0 && year == 0 {
		r.Vars.Alert = template.HTML("Invalid search: month cannot be specified if year is zero")
		return
	}
	if day > 0 && (year == 0 || month == 0) {
		r.Vars.Alert = template.HTML("Invalid search: day cannot be specified if month or year are zero")
		return
	}

	var key = &schema.Key{
		LCCN:  lccn,
		Year:  year,
		Month: month,
		Day:   day,
	}

	r.Issues = watcher.Scanner.LookupIssues(key)
	r.LCCN = lccn
	r.Year = year
	r.Month = month
	r.Day = day
}

// Render sets up the titles (and issues if relevant) for the template, then
// delegates to the base responder.Responder
func (r *resp) Render(t *tmpl.Template) {
	// Avoid any further work if we had an error
	if r.err != nil {
		r.Error(r.err.status, r.err.msg)
		return
	}

	// Copy all the typesafe data into our awful untyped structure
	r.Vars.Data["Titles"] = r.Titles
	r.Vars.Data["Issues"] = r.Issues
	r.Vars.Data["LCCN"] = r.LCCN
	r.Vars.Data["Year"] = r.Year
	r.Vars.Data["Month"] = r.Month
	r.Vars.Data["Day"] = r.Day
	r.Vars.Data["SearchAction"] = path.Join(basePath, "search")

	r.Responder.Render(t)
}
