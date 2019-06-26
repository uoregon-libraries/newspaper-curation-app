package issuefinderhandler

import (
	"fmt"
	"html/template"
	"path"
	"strconv"

	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
	"github.com/uoregon-libraries/newspaper-curation-app/src/web/webutil"
)

// Issue aliases a schema issue to add web-specific functionality
type Issue struct {
	*schema.Issue
}

func wrapIssues(list schema.IssueList) []*Issue {
	var out = make([]*Issue, len(list))
	for i, issue := range list {
		out[i] = &Issue{issue}
	}
	return out
}

// Link returns an "a" tag to view the issue, where applicable, or some
// explanatory text if no link is possible
func (i *Issue) Link() template.HTML {
	var contents, href string
	switch i.WorkflowStep {
	case schema.WSNil, schema.WSAwaitingProcessing, schema.WSUnfixableMetadataError:
		return template.HTML("N/A: not available")

	case schema.WSSFTP, schema.WSScan:
		return template.HTML("N/A: not in the system yet (needs to be queued)")

	case schema.WSInProduction:
		contents = "Production page list"
		href = path.Join(webutil.ProductionURL, "lccn", i.Title.LCCN, i.RawDate, "ed-"+strconv.Itoa(i.Edition))

	default:
		contents = "NCA Read-only page viewer"
		href = path.Join(webutil.FullPath("workflow", strconv.Itoa(i.DatabaseID), "view"))
	}

	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, href, contents))
}
