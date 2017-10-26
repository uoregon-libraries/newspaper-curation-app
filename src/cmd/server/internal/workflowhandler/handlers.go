package workflowhandler

import (
	"cmd/server/internal/responder"
	"config"
	"db"
	"fmt"
	"logger"
	"net/http"
	"path"
	"time"
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

	// Base path (desk view)
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(handle(canView(homeHandler)))

	// All other paths are centered around a specific issue
	var s2 = s.PathPrefix("/{issue_id}").Subrouter()

	// Claim / unclaim handlers are for both metadata and review
	s2.Path("/claim").Methods("POST").Handler(handle(canClaim(claimIssueHandler)))
	s2.Path("/unclaim").Methods("POST").Handler(handle(ownsIssue(unclaimIssueHandler)))

	// Alias for all the middleware we call to validate issue metadata entry:
	// - User has a role which allows entering metadata
	// - User owns the issue
	// - The issue is in the right workflow step
	var canEnterMetadata = func(f HandlerFunc) http.Handler {
		return handle(canWrite(ownsIssue(issueNeedsMetadataEntry(f))))
	}

	// Issue metadata paths
	s2.Path("/metadata").Handler(canEnterMetadata(enterMetadataHandler))
	s2.Path("/metadata/save").Methods("POST").Handler(canEnterMetadata(saveMetadataHandler))
	s2.Path("/page-numbering").Handler(canEnterMetadata(enterPageNumberHandler))
	s2.Path("/page-numbering/save").Methods("POST").Handler(canEnterMetadata(savePageNumberHandler))
	s2.Path("/queue").Methods("POST").Handler(canEnterMetadata(queuePageForReviewHandler))
	s2.Path("/unqueue").Methods("POST").Handler(canEnterMetadata(unqueuePageForReviewHandler))
	s2.Path("/report-error").Handler(canEnterMetadata(enterErrorHandler))
	s2.Path("/report-error/save").Methods("POST").Handler(canEnterMetadata(saveErrorHandler))

	// Alias for all the middleware we call to validate issue metadata review:
	// - User has a role which allows reviewing metadata
	// - User owns the issue
	// - The issue is in the right workflow step
	var canReviewMetadata = func(f HandlerFunc) http.Handler {
		return handle(canReview(ownsIssue(issueAwaitingMetadataReview(f))))
	}

	// Review paths
	var s3 = s2.PathPrefix("/review").Subrouter()
	s3.Path("/metadata").Handler(canReviewMetadata(reviewMetadataHandler))
	s3.Path("/page-numbering").Handler(canReviewMetadata(reviewPageNumbersHandler))
	s3.Path("/reject-form").Handler(canReviewMetadata(rejectIssueMetadataFormHandler))
	s3.Path("/reject").Methods("POST").Handler(canReviewMetadata(rejectIssueMetadataHandler))
	s3.Path("/approve").Methods("POST").Handler(canReviewMetadata(approveIssueMetadataHandler))

	Layout = responder.Layout.Clone()
	Layout.Path = path.Join(Layout.Path, "workflow")
	Layout.MustReadPartials("_mydeskissues.go.html")
	DeskTmpl = Layout.MustBuild("desk.go.html")
}

// homeHandler shows claimed workflow items that need to be finished as well as
// pending items which can be claimed
func homeHandler(resp *responder.Responder, i *Issue) {
	resp.Vars.Title = "Workflow"

	// Get issues currently on user's desk
	var uid = resp.Vars.User.ID
	var issues, err = db.FindIssuesOnDesk(uid)
	if err != nil {
		logger.Errorf("Unable to find issues on user %d's desk: %s", uid, err)
		resp.Vars.Alert = fmt.Sprintf("Unable to search for issues; contact support or try again later.")
		resp.Render(responder.Empty)
		return
	}
	resp.Vars.Data["MyDeskIssues"] = wrapDBIssues(issues)

	resp.Render(DeskTmpl)
}

func claimIssueHandler(resp *responder.Responder, i *Issue)              {}
func unclaimIssueHandler(resp *responder.Responder, i *Issue)            {}
func enterMetadataHandler(resp *responder.Responder, i *Issue)           {}
func saveMetadataHandler(resp *responder.Responder, i *Issue)            {}
func enterPageNumberHandler(resp *responder.Responder, i *Issue)         {}
func savePageNumberHandler(resp *responder.Responder, i *Issue)          {}
func queuePageForReviewHandler(resp *responder.Responder, i *Issue)      {}
func unqueuePageForReviewHandler(resp *responder.Responder, i *Issue)    {}
func reviewMetadataHandler(resp *responder.Responder, i *Issue)          {}
func reviewPageNumbersHandler(resp *responder.Responder, i *Issue)       {}
func rejectIssueMetadataFormHandler(resp *responder.Responder, i *Issue) {}
func rejectIssueMetadataHandler(resp *responder.Responder, i *Issue)     {}
func approveIssueMetadataHandler(resp *responder.Responder, i *Issue)    {}
func enterErrorHandler(resp *responder.Responder, i *Issue)              {}
func saveErrorHandler(resp *responder.Responder, i *Issue)               {}
