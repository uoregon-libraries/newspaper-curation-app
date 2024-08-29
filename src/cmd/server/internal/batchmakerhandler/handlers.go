package batchmakerhandler

import (
	"fmt"
	"html/template"
	"net/http"
	"path"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

var (
	basePath string

	// layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	layout *tmpl.TRoot

	// buildBatchFormTmpl is the form for finding issues to put into one or more batches
	buildBatchFormTmpl *tmpl.Template

	// showBatchIssuesFormTmpl shows the number of issues which will be put into
	// a batch and lets the user decide whether or not to proceed as well as
	// selecting the maximum batch size
	showBatchIssuesFormTmpl *tmpl.Template

	// showGenerateFormTmpl shows a form to preview what batches will be
	// generated and give the user a final chance to say "oops, no thanks"
	showGenerateFormTmpl *tmpl.Template
)

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string) {
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canBuild(buildBatchForm))
	s.Path("/filter").Handler(canBuild(showBatchIssuesForm))
	s.Path("/generate").Handler(canBuild(showGenerateForm))

	layout = responder.Layout.Clone()
	layout.Funcs(tmpl.FuncMap{
		"BatchMakerHomeURL":     func() string { return basePath },
		"BatchMakerFilterURL":   func() string { return path.Join(basePath, "filter") },
		"BatchMakerGenerateURL": func() string { return path.Join(basePath, "generate") },
	})
	layout.Path = path.Join(layout.Path, "batchmaker")

	buildBatchFormTmpl = layout.MustBuild("build.go.html")
	showBatchIssuesFormTmpl = layout.MustBuild("show-issues.go.html")
	showGenerateFormTmpl = layout.MustBuild("generate-form.go.html")
}

// filteredAggs returns a list of aggregations filtered by the form-submitted
// MOC ids, ready for use in handlers
func filteredAggs(req *http.Request) ([]*aggregation, error) {
	var err = req.ParseForm()
	if err != nil {
		return nil, fmt.Errorf("parsing form data: %w", err)
	}

	var list = req.Form["moc"]

	// If we got here with nothing selected, no sense hitting the potentially
	// expensive database view
	if len(list) == 0 {
		return nil, nil
	}

	var allAggs []*models.IssueAggregation
	allAggs, err = models.MOCIssueAggregations()
	if err != nil {
		return nil, fmt.Errorf("reading DB aggregations: %w", err)
	}

	var aggs []*models.IssueAggregation
	for _, val := range list {
		for _, agg := range allAggs {
			var id, _ = strconv.ParseInt(val, 10, 64)
			if agg.MOC.ID == id {
				aggs = append(aggs, agg)
			}
		}
	}

	return getAggregations(aggs)
}

// readAggs gets the responder, uses filteredAggs() to get the list of aggs,
// and automatically processes common errors or redirects needed for a handler.
// If exit is true, the caller should not process the request further.
func readAggs(w http.ResponseWriter, req *http.Request) (r *responder.Responder, aggs []*aggregation, exit bool) {
	var err error

	r = responder.Response(w, req)
	aggs, err = filteredAggs(req)
	if err != nil {
		logger.Errorf("Unable to get filtered aggregations list: %s", err)
		r.Error(http.StatusInternalServerError, "Error processing request - try again or contact support")
		return r, aggs, true
	}
	if len(aggs) == 0 {
		http.SetCookie(w, &http.Cookie{Name: "Alert", Value: "No selections made: nothing to batch", Path: "/"})
		http.Redirect(w, req, basePath, http.StatusFound)
		return r, aggs, true
	}

	return r, aggs, false
}

// buildBatchForm shows a form for filtering issues that are ready for batching
func buildBatchForm(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Filter Issues For Batching"

	var aggs, err = models.MOCIssueAggregations()
	if err != nil {
		logger.Errorf("Unable to load MOC issue aggregation: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to prepare lists - try again or contact support")
		return
	}

	r.Vars.Data["MOCIssueAggregations"], err = getAggregations(aggs)
	if err != nil {
		logger.Errorf("Unable to transform MOC aggregation data: %s", err)
		r.Error(http.StatusInternalServerError, "Error trying to prepare lists - try again or contact support")
		return
	}

	r.Render(buildBatchFormTmpl)
}

func renderBatchIssuesForm(r *responder.Responder, aggs []*aggregation) {
	r.Vars.Title = "Select Batch Parameters"
	r.Vars.Data["MaxPages"], _ = strconv.Atoi(r.Request.FormValue("maxpages"))
	r.Vars.Data["MOCIssueAggregations"] = aggs
	r.Render(showBatchIssuesFormTmpl)
}

// showBatchIssuesForm grabs issues for the selected MOCs and displays options
// for creating a batch
func showBatchIssuesForm(w http.ResponseWriter, req *http.Request) {
	var r, aggs, exit = readAggs(w, req)
	if exit {
		return
	}

	renderBatchIssuesForm(r, aggs)
}

func showGenerateForm(w http.ResponseWriter, req *http.Request) {
	var r, aggs, exit = readAggs(w, req)
	if exit {
		return
	}

	var max, _ = strconv.Atoi(req.FormValue("maxpages"))
	if max < 1 {
		r.Vars.Alert = template.HTML("Maximum size is invalid. Please enter a positive number.")
		renderBatchIssuesForm(r, aggs)
		return
	}

	// Build the (potentially final) batch queues, wrapping them to give the user
	// more context
	var queues []*Q
	for _, agg := range aggs {
		var readyQ = agg.ReadyForBatching
		var splitQs = readyQ.Split(max)
		var i int
		for _, sq := range splitQs {
			i++
			queues = append(queues, &Q{
				Sequence: i,
				MOC:      agg.MOC,
				Queue:    sq,
			})
		}
	}
	if len(queues) > 1 {
		r.Vars.Title = "Generate Batches?"
	} else {
		r.Vars.Title = "Generate Batch?"
	}
	r.Vars.Data["Queues"] = queues
	r.Vars.Data["MaxPages"] = max
	r.Vars.Data["MOCIssueAggregations"] = aggs

	r.Render(showGenerateFormTmpl)
}
