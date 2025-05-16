package issuefinderhandler

import (
	"fmt"
	"html/template"
	"net/url"
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

	case schema.WSInProduction, schema.WSAwaitingProdRemoval:
		contents = "Production page list"
		var u, err = url.Parse(webutil.ProductionURL)
		if err != nil {
			return template.HTML(fmt.Sprintf("Error: misconfigured production url %q: %s", webutil.ProductionURL, err))
		}
		u.Path = path.Join(u.Path, "lccn", i.Title.LCCN, i.RawDate, "ed-"+strconv.Itoa(i.Edition))
		href = u.String()

	default:
		contents = "NCA Read-only page viewer"
		href = path.Join(webutil.FullPath("workflow", strconv.FormatInt(i.DatabaseID, 10), "view"))
	}

	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, href, contents))
}
