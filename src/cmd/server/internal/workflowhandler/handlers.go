package workflowhandler

import (
	"cmd/server/internal/responder"
	"config"
	"db"
	"fmt"
	"html/template"
	"issuewatcher"
	"jobs"
	"net/http"
	"path"
	"schema"
	"user"
	"web/tmpl"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/gopkg/logger"
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

	// RejectIssueTmpl renders the view for reporting an issue which is rejected by the reviewer
	RejectIssueTmpl *tmpl.Template
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
	s3.Path("/reject-form").Handler(canReviewMetadata(rejectIssueMetadataFormHandler))
	s3.Path("/reject").Methods("POST").Handler(canReviewMetadata(rejectIssueMetadataHandler))
	s3.Path("/approve").Methods("POST").Handler(canReviewMetadata(approveIssueMetadataHandler))

	Layout = responder.Layout.Clone()
	Layout.Path = path.Join(Layout.Path, "workflow")
	Layout.MustReadPartials("_issue_table_rows.go.html", "_osdjs.go.html")
	DeskTmpl = Layout.MustBuild("desk.go.html")
	MetadataFormTmpl = Layout.MustBuild("metadata_form.go.html")
	ReportErrorTmpl = Layout.MustBuild("report_error.go.html")
	ReviewMetadataTmpl = Layout.MustBuild("metadata_review.go.html")
	RejectIssueTmpl = Layout.MustBuild("reject_issue.go.html")
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
		resp.Vars.Alert = template.HTML(fmt.Sprintf("Unable to search for issues; contact support or try again later."))
		resp.Render(responder.Empty)
		return
	}
	resp.Vars.Data["MyDeskIssues"] = wrapDBIssues(issues)

	// Get issues needing metadata
	issues, err = db.FindAvailableIssuesByWorkflowStep(schema.WSReadyForMetadataEntry)
	if err != nil {
		logger.Errorf("Unable to find issues needing metadata entry: %s", err)
		resp.Vars.Alert = template.HTML(fmt.Sprintf("Unable to search for issues; contact support or try again later."))
		resp.Writer.WriteHeader(http.StatusInternalServerError)
		resp.Render(responder.Empty)
		return
	}
	resp.Vars.Data["PendingMetadataIssues"] = wrapDBIssues(issues)

	// Get issues needing review which *weren't* queued by this user (unless the
	// user is allowed to self-review)
	issues, err = db.FindAvailableIssuesByWorkflowStep(schema.WSAwaitingMetadataReview)
	if err != nil {
		logger.Errorf("Unable to find issues needing metadata review: %s", err)
		resp.Vars.Alert = template.HTML(fmt.Sprintf("Unable to search for issues; contact support or try again later."))
		resp.Writer.WriteHeader(http.StatusInternalServerError)
		resp.Render(responder.Empty)
		return
	}
	var issuesTwo []*db.Issue
	for _, i := range issues {
		if i.MetadataEntryUserID != resp.Vars.User.ID || resp.Vars.User.PermittedTo(user.ReviewOwnMetadata) {
			issuesTwo = append(issuesTwo, i)
		}
	}
	resp.Vars.Data["PendingReviewIssues"] = wrapDBIssues(issuesTwo)

	resp.Render(DeskTmpl)
}

// claimIssueHandler just assigns the given issue to the logged-in user and
// sets a one-week expiration
func claimIssueHandler(resp *responder.Responder, i *Issue) {
	i.Claim(resp.Vars.User.ID)
	var err = i.Save()
	if err != nil {
		logger.Errorf("Unable to claim issue id %d by user %s: %s", i.ID, resp.Vars.User.Login, err)
		resp.Vars.Alert = template.HTML("Unable to claim issue; contact support or try again later.")
		resp.Writer.WriteHeader(http.StatusInternalServerError)
		resp.Render(responder.Empty)
		return
	}

	resp.Audit("claim", fmt.Sprintf("issue id %d", i.ID))
	http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Issue claimed successfully", Path: "/"})
	http.Redirect(resp.Writer, resp.Request, basePath, http.StatusFound)
}

// unclaimIssueHandler clears the issue's workflow data
func unclaimIssueHandler(resp *responder.Responder, i *Issue) {
	i.Unclaim()
	var err = i.Save()
	if err != nil {
		logger.Errorf("Unable to unclaim issue id %d for user %s: %s", i.ID, resp.Vars.User.Login, err)
		resp.Vars.Alert = template.HTML("Unable to unclaim issue; contact support or try again later.")
		resp.Writer.WriteHeader(http.StatusInternalServerError)
		resp.Render(responder.Empty)
		return
	}

	resp.Audit("unclaim", fmt.Sprintf("issue id %d", i.ID))
	http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Issue removed from your task list", Path: "/"})
	http.Redirect(resp.Writer, resp.Request, basePath, http.StatusFound)
}

// enterMetadataHandler shows the metadata entry form for the issue
func enterMetadataHandler(resp *responder.Responder, i *Issue) {
	resp.Vars.Title = "Issue Metadata / Page Numbers"
	resp.Vars.Data["Issue"] = i
	resp.Render(MetadataFormTmpl)
}

// saveMetadataHandler takes the form data, validates it, and on success
// updates the issue in the database
func saveMetadataHandler(resp *responder.Responder, i *Issue) {
	var changes = storeIssueMetadata(resp, i)
	var action = resp.Request.FormValue("action")

	switch action {
	case "autosave":
		autosave(resp, i, changes)
	case "savedraft":
		saveDraft(resp, i, changes)
	case "savequeue":
		saveQueue(resp, i, changes)
	default:
		logger.Warnf("Invalid action %q for saveMetadataHandler", action)
		resp.Writer.WriteHeader(http.StatusBadRequest)
		resp.Writer.Write([]byte("Bad Request"))
	}
}

// enterErrorHandler displays the form to enter an error for the given issue
func enterErrorHandler(resp *responder.Responder, i *Issue) {
	resp.Vars.Title = "Report Issue Error"
	resp.Vars.Data["Issue"] = i
	resp.Render(ReportErrorTmpl)
}

// saveErrorHandler records the error in the database, unclaims the issue, and
// flags it as needing admin attention
func saveErrorHandler(resp *responder.Responder, i *Issue) {
	i.Error = resp.Request.FormValue("error")
	if i.Error == "" {
		http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Error report empty; no action taken", Path: "/"})
		http.Redirect(resp.Writer, resp.Request, i.Path("metadata"), http.StatusFound)
		return
	}

	i.Unclaim()
	var err = i.Save()
	if err != nil {
		logger.Errorf("Unable to save issue id %d's error (POST: %#v): %s", i.ID, resp.Request.Form, err)
		resp.Vars.Alert = template.HTML("Error trying to save error report (no, the irony is not lost on us); try again or contact support")
		resp.Writer.WriteHeader(http.StatusInternalServerError)
		resp.Render(responder.Empty)
		return
	}

	resp.Audit("report-error", fmt.Sprintf("issue id %d", i.ID))
	http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Issue error reported", Path: "/"})
	http.Redirect(resp.Writer, resp.Request, basePath, http.StatusFound)
}

func reviewMetadataHandler(resp *responder.Responder, i *Issue) {
	resp.Vars.Title = "Reviewing Issue Metadata"
	resp.Vars.Data["Issue"] = i
	resp.Render(ReviewMetadataTmpl)
}

func approveIssueMetadataHandler(resp *responder.Responder, i *Issue) {
	// Validate the metadata again to be certain there were no last-minute
	// changes (e.g., database manipulation, out-of-band batch load, etc.)
	i.ValidateMetadata()
	if len(i.Errors()) > 0 {
		http.SetCookie(resp.Writer, &http.Cookie{Name: "Alert", Value: encodedErrors("approve", i.Errors()), Path: "/"})
		http.Redirect(resp.Writer, resp.Request, i.Path("review/metadata"), http.StatusFound)
		return
	}

	i.ApproveMetadata(resp.Vars.User.ID)
	var err = i.Save()
	if err != nil {
		logger.Errorf("Unable to save issue id %d's workflow approval by user %d (POST: %#v): %s",
			i.ID, resp.Vars.User.ID, resp.Request.Form, err)
		resp.Vars.Alert = template.HTML("Error trying to approve the issue; try again or contact support")
		resp.Writer.WriteHeader(http.StatusInternalServerError)
		resp.Render(responder.Empty)
		return
	}

	// We queue the issue finalization job, but whether it succeeds or not, the
	// issue was already successfully approved, so we just have to hope for the
	// best and log loudly if it doesn't work
	err = jobs.QueueFinalizeIssue(i.Issue, i.Location)
	if err != nil {
		logger.Criticalf("Unable to queue issue finalization for issue id %d: %s", i.ID, err)
	}
	resp.Audit("approve-metadata", fmt.Sprintf("issue id %d", i.ID))
	http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Issue approved", Path: "/"})
	http.Redirect(resp.Writer, resp.Request, basePath, http.StatusFound)
}

func rejectIssueMetadataFormHandler(resp *responder.Responder, i *Issue) {
	resp.Vars.Title = "Reject Issue"
	resp.Vars.Data["Issue"] = i
	resp.Render(RejectIssueTmpl)
}

func rejectIssueMetadataHandler(resp *responder.Responder, i *Issue) {
	var notes = resp.Request.FormValue("notes")
	if notes == "" {
		http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Rejection notes empty; no action taken", Path: "/"})
		http.Redirect(resp.Writer, resp.Request, i.Path("review/metadata"), http.StatusFound)
		return
	}

	i.RejectMetadata(resp.Vars.User.ID, notes)
	var err = i.Save()
	if err != nil {
		logger.Errorf("Unable to save issue id %d's rejection notes (POST: %#v): %s", i.ID, resp.Request.Form, err)
		resp.Vars.Alert = template.HTML("Error trying to save rejection notes; try again or contact support")
		resp.Writer.WriteHeader(http.StatusInternalServerError)
		resp.Render(responder.Empty)
		return
	}

	resp.Audit("reject-metadata", fmt.Sprintf("issue id %d", i.ID))
	http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Issue rejected", Path: "/"})
	http.Redirect(resp.Writer, resp.Request, basePath, http.StatusFound)
}
