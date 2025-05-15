package managelivehandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/duration"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

type jsonIssue struct {
	*models.FlatIssue
	FullTitle    string
	WentLiveAt   string
	LiveIssueURL string
	LiveTitleURL string
	LiveBatchURL string
	PublishedOn  string
}

const dateFmt = "Mon Jan _2, 2006"

func wrapIssue(i *models.FlatIssue) *jsonIssue {
	var ji = &jsonIssue{FlatIssue: i, PublishedOn: i.Date}
	var pubd, err = time.Parse("2006-01-02", i.Date)
	if err == nil {
		ji.PublishedOn = pubd.Format(dateFmt)
	}
	ji.WentLiveAt = i.WentLiveAt.Format(dateFmt)
	ji.FullTitle = fmt.Sprintf("%s (%s)", i.TitleName, i.LCCN)
	ji.LiveTitleURL = path.Join(conf.NewsWebroot, "lccn", i.LCCN)
	ji.LiveIssueURL = path.Join(ji.LiveTitleURL, i.Date, "ed-"+strconv.Itoa(i.Edition))
	ji.LiveBatchURL = path.Join(conf.NewsWebroot, "batches", i.BatchFullName)

	return ji
}

type jsonResponse struct {
	Code         int
	Message      string
	Issues       []*jsonIssue
	TotalResults uint64
}

type issueFilter func(val string) *models.FlatIssueFinder

func wentLiveFilter(f *models.FlatIssueFinder) issueFilter {
	return func(val string) *models.FlatIssueFinder {
		var d, err = duration.Parse(val)
		// Errors shouldn't be possible unless somebody hacks the form, so we
		// just pretend the "went live" filter wasn't there
		if err != nil {
			return f
		}
		var now = time.Now()
		var then = d.SubtractFromTime(now)
		return f.WentLiveBetween(then, now)
	}
}

// jsonHandler produces a JSON feed of issue information to enable
// rendering a subset of issues
func jsonHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var response, err = getJSONIssues(r)
	if err != nil {
		logger.Errorf("Unable to get JSON issues: %s", err)
		r.Writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(r.Writer, `{"Code": %d, "Message": %q}`, http.StatusInternalServerError, "Unable to retrieve issues from the database! Try again or contact support.")
		return
	}

	var data []byte
	data, err = json.Marshal(response)
	if err != nil {
		r.Writer.WriteHeader(http.StatusInternalServerError)
		logger.CriticalFixNeeded(fmt.Sprintf("Unable to marshal %#v", response), err)
		fmt.Fprintf(r.Writer, `{"Code": %d, "Message": %q}`, http.StatusInternalServerError, "Unable to retrieve issues from the database! Try again or contact support.")
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(response.Code)

	// Ignore the Write error here - a client disconnecting mid-write causes an
	// error which we do not care about
	_, _ = w.Write(data)
}

func getJSONIssues(resp *responder.Responder) (*jsonResponse, error) {
	var err error
	var response = &jsonResponse{Code: http.StatusOK}

	// Build filter function map to prepare our flat issue finder
	var finder = models.FlatIssues().Live()
	var filterMap = map[string]issueFilter{
		"moc":       finder.MOC,
		"lccn":      finder.LCCN,
		"pubdate":   finder.Date,
		"went-live": wentLiveFilter(finder),
	}

	// Apply filters based on request parameters
	resp.Request.ParseForm()
	for key, applyFilter := range filterMap {
		var value = resp.Request.FormValue(key)
		if value != "" {
			finder = applyFilter(value)
		}
	}

	response.TotalResults, err = finder.Count()
	if err != nil {
		logger.Errorf("Error counting issues in live-issue JSON handler: %s", err)
		return nil, err
	}

	var issues []*models.FlatIssue
	issues, err = finder.Limit(100).Fetch()
	if err != nil {
		logger.Errorf("Error reading issues in live-issue JSON handler: %s", err)
		return nil, err
	}

	response.Issues = make([]*jsonIssue, len(issues))
	for i, issue := range issues {
		response.Issues[i] = wrapIssue(issue)
	}

	return response, nil
}
