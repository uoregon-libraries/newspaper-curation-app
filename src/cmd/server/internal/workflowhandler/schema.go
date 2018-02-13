package workflowhandler

import (
	"db"
	"fmt"
	"html/template"
	"issuesearch"

	"path"
	"path/filepath"
	"schema"
	"strconv"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
)

// Issue wraps the DB issue, and decorates them with display-friendly functions
type Issue struct {
	*db.Issue
	si     *schema.Issue
	errors []string
}

func wrapDBIssue(dbIssue *db.Issue) *Issue {
	var si, err = dbIssue.SchemaIssue()

	// This shouldn't realistically happen, so we log and return nothing
	if err != nil {
		logger.Errorf("Unable to get schema.Issue for issue id %d: %s", dbIssue.ID, err)
		return nil
	}

	return &Issue{Issue: dbIssue, si: si}
}

func wrapDBIssues(dbIssues []*db.Issue) []*Issue {
	var list []*Issue
	for _, dbIssue := range dbIssues {
		var i = wrapDBIssue(dbIssue)
		if i == nil {
			return nil
		}
		list = append(list, i)
	}

	return list
}

// Title returns the issue's title's name
func (i *Issue) Title() string {
	return i.si.Title.Name
}

// LCCN returns the issue's title's LCCN
func (i *Issue) LCCN() string {
	return i.si.Title.LCCN
}

// Date returns the issue's raw date string
func (i *Issue) Date() string {
	return i.si.RawDate
}

// JP2Files aggregates all the JP2s that exist in this issue's directory
func (i *Issue) JP2Files() []string {
	var list []string

	if len(i.si.Files) == 0 {
		i.si.FindFiles()
	}

	for _, f := range i.si.Files {
		if strings.ToUpper(filepath.Ext(f.Location)) == ".JP2" {
			list = append(list, f.Location)
		}
	}

	return list
}

// TaskDescription returns a human-friendly explanation of the current place
// this issue is within the workflow
func (i *Issue) TaskDescription() string {
	switch i.WorkflowStep {
	case schema.WSAwaitingProcessing:
		return "Not yet entered into the workflow"

	case schema.WSAwaitingPageReview:
		return "Ready for page review (renaming files / validating raw PDFs / TIFFs)"

	case schema.WSReadyForMetadataEntry:
		return "Awaiting metadata entry / page numbering"

	case schema.WSAwaitingMetadataReview:
		return "Awaiting review (metadata and page numbers)"

	case schema.WSReadyForBatching:
		return "Ready to be built in a batch and loaded"

	default:
		logger.Criticalf("Invalid workflow step for issue %d: %q", i.ID, i.WorkflowStepString)
		return "UNKNOWN!"
	}
}

// WorkflowExpiration returns the date and time of "workflow expiration": when
// this item is no longer claimed by the workflow owner
func (i *Issue) WorkflowExpiration() string {
	return i.WorkflowOwnerExpiresAt.Format("2006-01-02 at 15:04")
}

// actionButton creates an action button wrapped by a one-off form for actions
// related to a single issue
func (i *Issue) actionButton(label, actionPath, classes string) template.HTML {
	return template.HTML(fmt.Sprintf(
		`<form action="%s" method="POST" class="actions"><button type="submit" class="btn %s">%s</button></form>`,
		i.Path(actionPath), classes, label))
}

// actionLink creates a link to the given action; for non-destructive actions
// like visiting a form page
func (i *Issue) actionLink(label, actionPath, classes string) template.HTML {
	return template.HTML(fmt.Sprintf(`<a href="%s" class="%s">%s</a>`, i.Path(actionPath), classes, label))
}

// IsOwned returns true if the owner ID is nonzero *and* the workflow owner
// expiration time has not passed
func (i *Issue) IsOwned() bool {
	return i.WorkflowOwnerID != 0 && time.Now().Before(i.WorkflowOwnerExpiresAt)
}

// Actions returns the action link HTML for each possible action the owner can
// take for this issue
func (i *Issue) Actions() []template.HTML {
	var actions []template.HTML

	if i.IsOwned() {
		switch i.WorkflowStep {
		case schema.WSReadyForMetadataEntry:
			actions = append(actions, i.actionLink("Edit", "metadata", ""))

		case schema.WSAwaitingMetadataReview:
			actions = append(actions, i.actionLink("Review", "review/metadata", ""))
		}

		actions = append(actions, i.actionButton("Unclaim", "/unclaim", "btn-danger"))
	} else {
		actions = append(actions, i.actionButton("Claim", "/claim", "btn-primary"))
	}

	return actions
}

// Path returns the path for any basic actions on this issue
func (i *Issue) Path(actionPath string) string {
	return path.Join(basePath, strconv.Itoa(i.ID), actionPath)
}

// ValidateMetadata checks all fields for validity and sets up i.Errors to
// describe anything wrong
func (i *Issue) ValidateMetadata() {
	i.errors = nil
	var addError = func(msg string) { i.errors = append(i.errors, msg) }
	var validDate = func(dtString, fieldName string) {
		var dtLayout = "2006-01-02"
		var dt, err = time.Parse(dtLayout, dtString)
		if err != nil || dt.Format(dtLayout) != dtString {
			addError(fmt.Sprintf("%q is not a valid date", fieldName))
		}
	}
	var notBlank = func(val, fieldName string) {
		if val == "" {
			addError(fmt.Sprintf("%q cannot be blank", fieldName))
		}
	}

	validDate(i.Issue.Date, "Issue Date")
	validDate(i.DateAsLabeled, "Date As Labeled")
	notBlank(i.Volume, "Volume Number")
	notBlank(i.Issue.Issue, "Issue Number")
	if i.Edition == 0 {
		addError(`"Edition Number" cannot be zero`)
	}

	var numLabels = len(i.PageLabels)
	var numFiles = len(i.JP2Files())
	if numLabels < numFiles {
		addError("Page labeling isn't completed")
	}
	if numLabels > numFiles {
		logger.Errorf("There are %d page labels (%#v), but only %d JP2 files!", numLabels, i.JP2Files(), numFiles)
		addError("Unknown error in page labeling; contact support or try again")
	}

	// Generate a new schema issue to test for dupes - database errors can be
	// logged, but not reported back to the user, so there's a lot of "log +
	// return" sadness below
	var err error
	i.si, err = i.Issue.SchemaIssue()
	if err != nil {
		logger.Criticalf("Unable to recreate schema.Issue for issue id %d: %s", i.ID, err)
		return
	}

	var list []*db.Issue
	list, err = db.FindIssuesByKey(i.si.Key())
	if err != nil {
		logger.Criticalf("Unable to search for database issue %q: %s", i.si.Key(), err)
		return
	}
	for _, issue := range list {
		if issue.ID != i.ID {
			addError("This is a duplicate of another issue; double-check the date and edition number, or contact support")
		}
	}

	// Now check for live dupes - given we're generating a search key from a real
	// issue, we can safely ignore the ParseSearchKey error
	var key, _ = issuesearch.ParseSearchKey(i.si.Key())
	var schemaIssues = watcher.Scanner.LookupIssues(key)
	for _, issue := range schemaIssues {
		if issue.WorkflowStep == schema.WSInProduction {
			addError(fmt.Sprintf("This is a duplicate of a live issue (in %q); "+
				"double-check the date and edition number, or contact support",
				issue.Batch.Fullname()))
		}
	}
}

// Errors returns validation errors
func (i *Issue) Errors() []string {
	return i.errors
}
