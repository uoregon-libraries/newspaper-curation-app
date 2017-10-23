package workflowhandler

import (
	"cmd/server/internal/responder"
	"config"
	"db"
	"fmt"
	"logger"
	"net/http"
	"path"
	"strconv"
	"user"
	"web/tmpl"

	"github.com/gorilla/mux"
)

var (
	conf *config.Config

	// basePath is the path to the main workflow page.  Subpages all start with this path.
	basePath string

	// Layout is the base template, cloned from the responder's layout, from
	// which all workflow pages are built
	Layout *tmpl.TRoot

	// DeskTmpl renders the main "workflow desk" page
	DeskTmpl *tmpl.Template

	// PendingTmpl renders the list of issues which need metadata entry or metadata review and aren't yet owned
	PendingTmpl *tmpl.Template

	// MetadataFormTmpl renders the form for entering metadata for an issue
	MetadataFormTmpl *tmpl.Template

	// PageLabelFormTmpl renders the form for entering page numbers for an issue
	PageLabelFormTmpl *tmpl.Template

	// ReportErrorTmpl renders the form for reporting errors on an issue
	ReportErrorTmpl *tmpl.Template

	// ReviewMetadataTmpl renders the view for reviewing metadata
	ReviewMetadataTmpl *tmpl.Template

	// ReviewPageLabelsTmpl renders the view for reviewing page labels
	ReviewPageLabelsTmpl *tmpl.Template

	// RejectIssueTmpl renders the view for reporting an issue which is rejected by the reviewer
	RejectIssueTmpl *tmpl.Template
)

// Setup sets up all the workflow-specific routing rules and does any other
// init necessary for workflow handling
func Setup(r *mux.Router, webPath string, c *config.Config) {
	conf = c
	basePath = webPath

	// Alias the permission checks
	var canView = func(h http.HandlerFunc) http.Handler {
		return responder.MustHavePrivilege(user.ViewMetadataWorkflow, h)
	}
	var canWrite = func(h http.HandlerFunc) http.Handler {
		return responder.MustHavePrivilege(user.EnterIssueMetadata, h)
	}
	var canReview = func(h http.HandlerFunc) http.Handler {
		return responder.MustHavePrivilege(user.ReviewIssueMetadata, h)
	}

	// Base path (desk view)
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(canView(homeHandler))

	// Issue metadata paths
	var s2 = s.PathPrefix("/{issue_id}").Subrouter()
	s2.Path("/claim").Methods("POST").Handler(canWrite(claimIssueHandler))
	s2.Path("/unclaim").Methods("POST").Handler(canWrite(unclaimIssueHandler))
	s2.Path("/metadata").Handler(canWrite(enterMetadataHandler))
	s2.Path("/metadata/save").Methods("POST").Handler(canWrite(saveMetadataHandler))
	s2.Path("/page-numbering").Handler(canWrite(enterPageNumberHandler))
	s2.Path("/page-numbering/save").Methods("POST").Handler(canWrite(savePageNumberHandler))
	s2.Path("/queue").Methods("POST").Handler(canWrite(queuePageForReviewHandler))
	s2.Path("/unqueue").Methods("POST").Handler(canWrite(unqueuePageForReviewHandler))
	s2.Path("/report-error").Handler(canWrite(enterErrorHandler))
	s2.Path("/report-error/save").Methods("POST").Handler(canWrite(saveErrorHandler))

	// Review paths
	var s3 = s2.PathPrefix("/review").Subrouter()
	s3.Path("/metadata").Handler(canReview(reviewMetadataHandler))
	s3.Path("/page-numbering").Handler(canReview(reviewPageNumbersHandler))
	s3.Path("/reject-form").Handler(canReview(rejectIssueMetadataFormHandler))
	s3.Path("/reject").Methods("POST").Handler(canReview(rejectIssueMetadataHandler))
	s3.Path("/approve").Methods("POST").Handler(canReview(approveIssueMetadataHandler))

	Layout = responder.Layout.Clone()
	Layout.Path = path.Join(Layout.Path, "workflow")
	Layout.MustReadPartials("_mydeskissues.go.html")
	DeskTmpl = Layout.MustBuild("desk.go.html")
}

// findIssue attempts to load the issue specified in the request's issue id
// parameter, and returns it.  If there is no issue for the given id, nil is
// returned and the caller should do nothing, as http headers / rendering is
// already done.
func findIssue(r *responder.Responder) *db.Issue {
	var idStr = mux.Vars(r.Request)["issue_id"]
	var id, _ = strconv.Atoi(idStr)
	if id == 0 {
		r.Vars.Alert = fmt.Sprintf("Invalid issue id %#v", idStr)
		r.Render(responder.Empty)
		return nil
	}

	var i, err = db.FindIssue(id)
	if err != nil {
		logger.Errorf("Error trying to find issue id %d: %s", id, err)
		r.Vars.Alert = fmt.Sprintf("Unable to find issue id %d", id)
		r.Render(responder.Empty)
		return nil
	}
	if i == nil {
		r.Vars.Alert = fmt.Sprintf("Unable to find issue id %d", id)
		r.Render(responder.Empty)
		return nil
	}

	return i
}

// homeHandler shows claimed workflow items that need to be finished as well as
// pending items which can be claimed
func homeHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Vars.Title = "Workflow"

	// Get issues currently on user's desk
	var uid = r.Vars.User.ID
	var issues, err = db.FindIssuesOnDesk(uid)
	if err != nil {
		logger.Errorf("Unable to find issues on user %d's desk: %s", uid, err)
		r.Vars.Alert = fmt.Sprintf("Unable to search for issues; contact support or try again later.")
		r.Render(responder.Empty)
		return
	}
	r.Vars.Data["MyDeskIssues"] = wrapDBIssues(issues)

	r.Render(DeskTmpl)
}

func claimIssueHandler(w http.ResponseWriter, req *http.Request)              {}
func unclaimIssueHandler(w http.ResponseWriter, req *http.Request)            {}
func enterMetadataHandler(w http.ResponseWriter, req *http.Request)           {}
func saveMetadataHandler(w http.ResponseWriter, req *http.Request)            {}
func enterPageNumberHandler(w http.ResponseWriter, req *http.Request)         {}
func savePageNumberHandler(w http.ResponseWriter, req *http.Request)          {}
func queuePageForReviewHandler(w http.ResponseWriter, req *http.Request)      {}
func unqueuePageForReviewHandler(w http.ResponseWriter, req *http.Request)    {}
func reviewMetadataHandler(w http.ResponseWriter, req *http.Request)          {}
func reviewPageNumbersHandler(w http.ResponseWriter, req *http.Request)       {}
func rejectIssueMetadataFormHandler(w http.ResponseWriter, req *http.Request) {}
func rejectIssueMetadataHandler(w http.ResponseWriter, req *http.Request)     {}
func approveIssueMetadataHandler(w http.ResponseWriter, req *http.Request)    {}
func enterErrorHandler(w http.ResponseWriter, req *http.Request)              {}
func saveErrorHandler(w http.ResponseWriter, req *http.Request)               {}
