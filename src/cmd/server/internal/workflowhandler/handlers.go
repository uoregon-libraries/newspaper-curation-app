package workflowhandler

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// searchIssueError handles generic response logic for database errors which
// can occur when searching for issues
func searchIssueError(resp *responder.Responder) {
	resp.Vars.Alert = template.HTML(fmt.Sprintf("Unable to search for issues; contact support or try again later."))
	resp.Writer.WriteHeader(http.StatusInternalServerError)
	resp.Render(responder.Empty)
}

// homeHandler shows claimed workflow items that need to be finished as well as
// pending items which can be claimed
func homeHandler(resp *responder.Responder, i *Issue) {
	resp.Vars.Title = "Workflow"

	var err error
	resp.Vars.Data["DeskCount"], err = models.Issues().OnDesk(resp.Vars.User.ID).Count()
	if err == nil {
		resp.Vars.Data["CurateCount"], err = models.Issues().Available().InWorkflowStep(schema.WSReadyForMetadataEntry).Count()
	}
	if err == nil {
		resp.Vars.Data["ReviewCount"], err = models.Issues().Available().InWorkflowStep(schema.WSAwaitingMetadataReview).Count()
	}
	if err == nil {
		resp.Vars.Data["ErrorCount"], err = models.Issues().Available().InWorkflowStep(schema.WSUnfixableMetadataError).Count()
	}
	if err != nil {
		logger.Errorf("Unable to count issues for workflow homepage: %s", err)
		searchIssueError(resp)
		return
	}

	resp.Render(DeskTmpl)
}

// viewIssueHandler displays the given issue to the user so it can be looked
// over without having to claim it
func viewIssueHandler(resp *responder.Responder, i *Issue) {
	i.ValidateMetadata()
	resp.Vars.Title = "Issue Metadata / Page Numbers"
	resp.Vars.Data["Issue"] = i
	resp.Render(ViewIssueTmpl)
}

// claimIssueHandler just assigns the given issue to the logged-in user and
// sets a one-week expiration
func claimIssueHandler(resp *responder.Responder, i *Issue) {
	var err = i.Claim(resp.Vars.User.ID)
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
	var err = i.Unclaim(resp.Vars.User.ID)
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
	i.ValidateMetadata()
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

func reviewMetadataHandler(resp *responder.Responder, i *Issue) {
	i.ValidateMetadata()
	resp.Vars.Title = "Reviewing Issue Metadata"
	resp.Vars.Data["Issue"] = i
	resp.Render(ReviewMetadataTmpl)
}

func approveIssueMetadataHandler(resp *responder.Responder, i *Issue) {
	// Validate the metadata again to be certain there were no last-minute
	// changes (e.g., database manipulation, out-of-band batch load, etc.)
	i.ValidateMetadata()
	if i.Errors().Major().Len() > 0 {
		http.SetCookie(resp.Writer, &http.Cookie{Name: "Alert", Value: encodedErrors("approve", i.Errors()), Path: "/"})
		http.Redirect(resp.Writer, resp.Request, i.Path("review/metadata"), http.StatusFound)
		return
	}

	var err = i.ApproveMetadata(resp.Vars.User.ID)
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
	err = jobs.QueueFinalizeIssue(i.Issue)
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

	var err = i.RejectMetadata(resp.Vars.User.ID, notes)
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
