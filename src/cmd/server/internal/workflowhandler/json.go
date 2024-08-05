package workflowhandler

import (
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/humanize"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// Action describes the path and text for anything a user is allowed to do with
// an issue
type Action struct {
	Text string
	Path string
	Type string // "link", "button", or "button-danger"
}

// JSONIssue stores raw data needed to list an issue on the main
// desk/workflow-tab page. All fields are pre-computed so that a request for
// issue data simply returns lists of immediately usable JSON data.
type JSONIssue struct {
	Title      string
	LCCN       string
	Date       string
	PageCount  int
	Task       string
	Expiration string
	Waiting    string // How long since this issue's metadata was entered
	Actions    []Action
}

func jsonify(dbIssues []*models.Issue, user *models.User) []*JSONIssue {
	var list []*JSONIssue
	for _, dbIssue := range dbIssues {
		var i = wrapDBIssue(dbIssue)
		if i == nil {
			return nil
		}
		list = append(list, wrapJSON(i, user))
	}

	return list
}

func wrapJSON(i *Issue, u *models.User) *JSONIssue {
	var ji = &JSONIssue{
		Title:      i.Title(),
		LCCN:       i.LCCN(),
		Date:       i.Date(),
		PageCount:  i.PageCount,
		Task:       i.TaskDescription(),
		Expiration: i.WorkflowExpiration(),
		Waiting:    humanize.Duration(time.Since(i.MetadataEnteredAt)),
	}

	var addAction = func(text, subpath string, atype string) {
		ji.Actions = append(ji.Actions, Action{text, i.Path(subpath), atype})
	}

	// Everybody can view any issues we're willing to list
	addAction("View", "view", "link")

	// Add permission-based actions
	var can = Can(u)
	if can.EnterMetadata(i) {
		addAction("Edit", "metadata", "link")
	}
	if can.ReviewMetadata(i) {
		addAction("Review", "review/metadata", "link")
	}
	if can.ReviewUnfixable(i) {
		addAction("Review", "errors/view", "link")
	}
	if can.Claim(i) {
		addAction("Claim", "claim", "button")
	}
	if can.Unclaim(i) {
		addAction("Unclaim", "unclaim", "button-danger")
	}

	return ji
}
