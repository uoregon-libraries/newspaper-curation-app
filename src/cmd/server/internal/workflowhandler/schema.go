package workflowhandler

import (
	"db"
	"fmt"
	"html/template"
	"logger"
	"schema"
)

// Issue wraps the DB issue, and decorates them with display-friendly functions
type Issue struct {
	*db.Issue
	si *schema.Issue
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

// Name returns a human-friendly representation of the issue
func (i *Issue) Name() string {
	return fmt.Sprintf("%s, %s", i.si.Title.Name, i.si.DateStringReadable())
}

// TaskDescription returns a human-friendly explanation of the current place
// this issue is within the workflow
func (i *Issue) TaskDescription() string {
	switch i.WorkflowStep {
	case db.WSAwaitingPageReview:
		return "Ready for page review (renaming files / validating raw PDFs / TIFFs)"

	case db.WSReadyForMetadataEntry:
		return "Awaiting metadata entry / page numbering"

	case db.WSAwaitingMetadataReview:
		return "Awaiting review (metadata and page numbers)"

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
		`<form action="/%d/%s" method="POST"><button type="submit" class="btn %s">"%s"</button></form>`,
		i.ID, actionPath, classes, label))
}

// Actions returns the action link HTML for each possible action the owner can
// take for this issue
func (i *Issue) Actions() []template.HTML {
	var actions []template.HTML

	switch i.WorkflowStep {
	case db.WSReadyForMetadataEntry:
		actions = append(actions, i.actionButton("Metadata", "metadata", "btn-default"))
		actions = append(actions, i.actionButton("Page Numbering", "page-numbering", "btn-default"))

	case db.WSAwaitingMetadataReview:
		actions = append(actions, i.actionButton("Metadata", "review/metadata", "btn-default"))
		actions = append(actions, i.actionButton("Page Numbering", "review/page-numbering", "btn-default"))
	}

	actions = append(actions, i.actionButton("Unclaim", "/unclaim", ""))

	return actions
}
