package managelivehandler

import (
	"fmt"
	"net/http"
	"path"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

var (
	basePath string
	conf     *config.Config

	// Layout is the base template, cloned from the responder's layout, from
	// which all subpages are built
	layout *tmpl.TRoot

	// findIssuesTmpl is the form for searching for issues that need to be pulled
	// from prod
	findIssuesTmpl *tmpl.Template

	// queueRemovalTmpl is the form for confirming and explaining live issue removal
	queueRemovalTmpl *tmpl.Template
)

// canFlagIssues verifies the user is allowed to use this part of NCA in the
// first place. The search form wouldn't be dangerous for an unauthorized user,
// but it serves no purpose compared to the other (and broader) issue search.
func canFlagIssues(h http.HandlerFunc) http.Handler {
	return responder.MustHavePrivilege(privilege.FlagLiveIssues, h)
}

// Setup sets up all the routing rules and other configuration
func Setup(r *mux.Router, baseWebPath string, c *config.Config) {
	basePath = baseWebPath
	conf = c

	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canFlagIssues(buildIssueFindForm))
	s.Path("/search").Handler(canFlagIssues(jsonHandler))
	s.Path("/queue-issue-removal").Handler(canFlagIssues(buildIssueQueueRemovalForm))

	layout = responder.Layout.Clone()
	layout.Path = path.Join(layout.Path, "manage-live-issues")
	findIssuesTmpl = layout.MustBuild("find-issues.go.html")
	queueRemovalTmpl = layout.MustBuild("queue-removal.go.html")
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
	r.Vars.Data["QueueRemovalURL"] = path.Join(basePath, "queue-issue-removal")
	r.Render(findIssuesTmpl)
}

func buildIssueQueueRemovalForm(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Queue Live Issue For Removal"

	r.Request.ParseForm()
	var idstr = r.Request.FormValue("id")
	var id, _ = strconv.ParseInt(idstr, 10, 64)
	if id < 1 {
		r.Error(http.StatusInternalServerError, fmt.Sprintf("Unable to load issue: %q is not a valid id.", idstr))
		return
	}

	var err error
	r.Vars.Data["Issue"], err = models.FindIssue(id)
	if err != nil {
		logger.Errorf("Unable to load issue for live-issue rejection form: %s", err)
		r.Error(http.StatusInternalServerError, "Unable to load issue. Try again or contact support.")
		return
	}

	r.Vars.Data["QueueURL"] = path.Join(basePath, "queue-issue-removal")
	r.Render(queueRemovalTmpl)
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
