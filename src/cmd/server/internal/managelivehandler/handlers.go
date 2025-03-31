package managelivehandler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

var (
	basePath string

	// Layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	layout *tmpl.TRoot

	// findIssuesTmpl is the form for searching for issues that need to be pulled
	// from prod
	findIssuesTmpl *tmpl.Template
)

// canFlagIssues verifies the user is allowed to use this part of NCA in the
// first place. The search form wouldn't be dangerous for an unauthorized user,
// but it serves no purpose compared to the other (and broader) issue search.
func canFlagIssues(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.FlagLiveIssues, h)
}

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string) {
	basePath = baseWebPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canFlagIssues(buildIssueFindForm))
	s.Path("/search").Handler(canFlagIssues(jsonHandler))

	layout = responder.Layout.Clone()
	layout.Path = path.Join(layout.Path, "manage-live-issues")
	findIssuesTmpl = layout.MustBuild("find-issues.go.html")
}

// buildIssueFindForm spits out the search form
func buildIssueFindForm(w http.ResponseWriter, req *http.Request) {
	var err error
	var r = responder.Response(w, req)
	r.Vars.Title = "Find Live Issues"
	r.Vars.Data["Titles"], err = allTitles()
	if err != nil {
		logger.Errorf("Unable to load titles for issue search form: %s", err)
		r.Error(http.StatusInternalServerError, "Unable to load titles. Try again or contact support.")
		return
	}

	r.Vars.Data["MOCs"], err = models.AllMOCs()
	if err != nil {
		logger.Errorf("Unable to load MOCs for issue search form: %s", err)
		r.Error(http.StatusInternalServerError, "Unable to load MARC org codes. Try again or contact support.")
		return
	}

	r.Vars.Data["SearchURL"] = path.Join(basePath, "search")
	r.Render(findIssuesTmpl)
}

// allTitles returns a list of titles we want users to have as filter options
func allTitles() (titles schema.TitleList, err error) {
	var dbt models.TitleList
	dbt, err = models.Titles()
	if err != nil {
		return nil, err
	}

	for _, t := range dbt {
		titles = append(titles, t.SchemaTitle())
	}
	titles.SortByName()

	return titles, nil
}

// jsonHandler produces a JSON feed of issue information to enable
// rendering a subset of issues
func jsonHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var response, err = getJSONIssues(r)
	if err != nil {
		r.Writer.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(r.Writer, `{"code": %d, "message": %q}`, http.StatusInternalServerError, "Unable to retrieve issues from the database! Try again or contact support.")
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(response.Code)
	var data []byte
	data, err = json.Marshal(response)
	if err != nil {
		logger.Criticalf("Unable to marshal %#v: %s", response, err)
	}

	// Ignore the Write error here - a client disconnecting mid-write causes an
	// error which we do not care about
	_, _ = w.Write(data)
}

type jsonResponse struct {
	Code         int
	Message      string
	Issues       []*models.FlatIssue
	TotalResults uint64
}

func getJSONIssues(resp *responder.Responder) (*jsonResponse, error) {
	var err error
	var response = &jsonResponse{Code: http.StatusOK}

	// Get filters to prepare our flat issue finder
	var finder = models.FlatIssues().Live()
	var moc = resp.Request.FormValue("moc")
	if moc != "" {
		finder.MOC(moc)
	}
	var lccn = resp.Request.FormValue("lccn")
	if lccn != "" {
		finder.LCCN(lccn)
	}

	response.TotalResults, err = finder.Count()
	if err != nil {
		logger.Errorf("Error counting issues in live-issue JSON handler: %s", err)
		return nil, err
	}

	response.Issues, err = finder.Limit(100).Fetch()
	if err != nil {
		logger.Errorf("Error reading issues in live-issue JSON handler: %s", err)
		return nil, err
	}

	return response, nil
}
