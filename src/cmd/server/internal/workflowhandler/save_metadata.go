package workflowhandler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cmd/server/internal/responder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// storeIssueMetadata centralizes the logic for storing a metadata form's data
// and returning the list of changed fields
func storeIssueMetadata(resp *responder.Responder, i *Issue) map[string]string {
	// Set all fields and record changes for auditing / error logging
	var changes = make(map[string]string)
	var save = func(key string, store *string) {
		var val = resp.Request.FormValue(key)
		if val != *store {
			*store = val
			changes[key] = val
		}
	}

	save("issue_number", &i.Issue.Issue)
	save("edition_label", &i.EditionLabel)
	save("date_as_labeled", &i.DateAsLabeled)
	save("date", &i.Issue.Date)
	save("volume_number", &i.Volume)
	save("page_labels_csv", &i.PageLabelsCSV)
	save("draft_comment", &i.DraftComment)

	var key = "edition_number"
	var val = resp.Request.FormValue(key)
	var valNum, _ = strconv.Atoi(val)
	if i.Edition != valNum {
		i.Edition = valNum
		changes[key] = val
	}

	// Look for warning ignore/acceptance
	val = resp.Request.FormValue("ignore_warnings")
	logger.Warnf("val: %q", val)
	valNum, _ = strconv.Atoi(val)
	if valNum == i.ID {
		i.acceptWarnings = true
	}

	// This one's funny - we have to "deserialize" the label csv since the real
	// structure isn't what we get from the web
	i.PageLabels = strings.Split(i.PageLabelsCSV, ",")

	return changes
}

// saveIssue tries to store the issue to the database and returns the
// Issue.Save() response.  The caller doesn't need to log anything or set the
// http status on errors, as that is handled here.
func saveIssue(resp *responder.Responder, i *Issue, changes map[string]string) (ok bool) {
	// Don't bother saving to the database if nothing has changed
	if len(changes) == 0 {
		return true
	}

	var info = fmt.Sprintf("issue id %d (POST: %#v; Changes: %#v)", i.ID, resp.Request.Form, changes)
	var err = i.SaveWithoutAction()
	if err != nil {
		logger.Errorf("Unable to save metadata for %s: %s", info, err)
		resp.Writer.WriteHeader(http.StatusInternalServerError)
		return false
	}

	var auditAction = models.AuditActionFromString(resp.Request.FormValue("action"))
	resp.Audit(auditAction, info)
	return true
}

func autosave(resp *responder.Responder, i *Issue, changes map[string]string) {
	if ok := saveIssue(resp, i, changes); !ok {
		resp.Writer.Write([]byte("Internal Server Error"))
		return
	}
	resp.Writer.Write([]byte("OK"))
}

func saveDraft(resp *responder.Responder, i *Issue, changes map[string]string) {
	if ok := saveIssue(resp, i, changes); !ok {
		resp.Vars.Alert = "Unable to save issue; try again or contact support"
		enterMetadataHandler(resp, i)
		return
	}
	http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Saved Metadata", Path: "/"})
	http.Redirect(resp.Writer, resp.Request, i.Path("metadata"), http.StatusFound)
	return
}

func saveQueue(resp *responder.Responder, i *Issue, changes map[string]string) {
	// Save the metadata changes, if any; we want this stuff preserved regardless
	// of errors from invalid metadata
	if ok := saveIssue(resp, i, changes); !ok {
		resp.Vars.Alert = "Unable to save issue; try again or contact support"
		enterMetadataHandler(resp, i)
		return
	}

	// Check for metadata errors (which implicitly validates the metadata).  If
	// there are errors, let the user know and redisplay the form; we still keep
	// the saved changes in order to avoid losing metadata
	if i.Errors().Major().Len() > 0 {
		http.SetCookie(resp.Writer, &http.Cookie{Name: "Alert", Value: encodedErrors("queue", i.Errors().Major()), Path: "/"})
		http.Redirect(resp.Writer, resp.Request, i.Path("metadata"), http.StatusFound)
		return
	}

	// If we had no errors, but there are warnings, the user must explicitly say
	// they're okay queueing anyway
	if i.Errors().Minor().Len() > 0 && !i.acceptWarnings {
		http.SetCookie(resp.Writer, &http.Cookie{Name: "Alert", Value: "Warnings are present and must be remediated or skipped before queueing", Path: "/"})
		http.Redirect(resp.Writer, resp.Request, i.Path("metadata"), http.StatusFound)
		return
	}

	// Metadata is good: queue for review
	var err = i.QueueForMetadataReview(resp.Vars.User.ID)
	if err != nil {
		resp.Vars.Alert = "Unable to save issue; try again or contact support"
		enterMetadataHandler(resp, i)
		return
	}

	// If there were warnings, we want to note that the user has explicitly
	// chosen to ignore them
	if i.acceptWarnings {
		var warns []string
		for _, e := range i.Errors().Minor().All() {
			warns = append(warns, e.Message())
		}
		i.Save(models.ActionTypeInternalProcess, models.SystemUser.ID,
			fmt.Sprintf("ignoring warnings (approved by %q):\n\n%s", resp.Vars.User.Login, strings.Join(warns, "\n")))
	}

	resp.Audit(models.AuditActionQueueForReview, fmt.Sprintf("issue id %d", i.ID))
	http.SetCookie(resp.Writer, &http.Cookie{Name: "Info", Value: "Issue queued for review", Path: "/"})
	http.Redirect(resp.Writer, resp.Request, basePath, http.StatusFound)
}
