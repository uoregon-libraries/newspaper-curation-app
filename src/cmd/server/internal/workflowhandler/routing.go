package workflowhandler

import (
	"path"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/tmpl"
)

var (
	conf *config.Config

	// basePath is the path to the main workflow page.  Subpages all start with this path.
	basePath string

	// watcher is used to look for dupes when queueing an issue for review
	watcher *issuewatcher.Watcher

	// Layout is the base template, cloned from the responder's layout, from
	// which all workflow pages are built
	Layout *tmpl.TRoot

	// DeskTmpl renders the main "workflow desk" page
	DeskTmpl *tmpl.Template

	// MetadataFormTmpl renders the form for entering metadata for an issue
	MetadataFormTmpl *tmpl.Template

	// ReportErrorTmpl renders the form for reporting errors on an issue
	ReportErrorTmpl *tmpl.Template

	// ReviewMetadataTmpl renders the view for reviewing metadata
	ReviewMetadataTmpl *tmpl.Template

	// ViewErrorTmpl renders the view for deciding what to do with "unfixable" issues
	ViewErrorTmpl *tmpl.Template

	// RejectIssueTmpl renders the view for reporting an issue which is rejected by the reviewer
	RejectIssueTmpl *tmpl.Template

	// ViewIssueTmpl renders a read-only display of an issue
	ViewIssueTmpl *tmpl.Template
)

// Setup sets up all the workflow-specific routing rules and does any other
// init necessary for workflow handling
func Setup(r *mux.Router, webPath string, c *config.Config, w *issuewatcher.Watcher) {
	conf = c
	basePath = webPath
	watcher = w

	// Base path (desk view)
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(handle(canView(homeHandler)))

	// All other paths are centered around a specific issue
	var s2 = s.PathPrefix("/{issue_id}").Subrouter()

	// "Hidden" viewer path
	s2.Path("/view").Handler(handle(canView(viewIssueHandler)))

	// Claim / unclaim handlers are for both metadata and review
	s2.Path("/claim").Methods("POST").Handler(handle(canClaim(claimIssueHandler)))
	s2.Path("/unclaim").Methods("POST").Handler(handle(canUnclaim(unclaimIssueHandler)))

	// Issue metadata paths
	s2.Path("/metadata").Handler(handle(canEnterMetadata(enterMetadataHandler)))
	s2.Path("/metadata/save").Methods("POST").Handler(handle(canEnterMetadata(saveMetadataHandler)))
	s2.Path("/report-error").Handler(handle(canEnterMetadata(enterErrorHandler)))
	s2.Path("/report-error/save").Methods("POST").Handler(handle(canEnterMetadata(saveErrorHandler)))

	// Review paths
	var s3 = s2.PathPrefix("/review").Subrouter()
	s3.Path("/metadata").Handler(handle(canReviewMetadata(reviewMetadataHandler)))
	s3.Path("/reject-form").Handler(handle(canReviewMetadata(rejectIssueMetadataFormHandler)))
	s3.Path("/reject").Methods("POST").Handler(handle(canReviewMetadata(rejectIssueMetadataHandler)))
	s3.Path("/approve").Methods("POST").Handler(handle(canReviewMetadata(approveIssueMetadataHandler)))

	// Error review paths
	var s4 = s2.PathPrefix("/errors").Subrouter()
	s4.Path("/view").Handler(handle(canReviewUnfixable(reviewUnfixableHandler)))
	s4.Path("/save").Methods("POST").Handler(handle(canReviewUnfixable(saveUnfixableHandler)))

	Layout = responder.Layout.Clone()
	Layout.Funcs(tmpl.FuncMap{"Can": Can})
	Layout.Path = path.Join(Layout.Path, "workflow")
	Layout.MustReadPartials("_issue_table_rows.go.html", "_osdjs.go.html", "_view_issue.go.html")
	DeskTmpl = Layout.MustBuild("desk.go.html")
	MetadataFormTmpl = Layout.MustBuild("metadata_form.go.html")
	ReportErrorTmpl = Layout.MustBuild("report_error.go.html")
	ReviewMetadataTmpl = Layout.MustBuild("metadata_review.go.html")
	ViewErrorTmpl = Layout.MustBuild("error_review.go.html")
	RejectIssueTmpl = Layout.MustBuild("reject_issue.go.html")
	ViewIssueTmpl = Layout.MustBuild("view_issue.go.html")
}
