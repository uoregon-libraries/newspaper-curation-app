package batchmakerhandler

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/issuequeue"
	"github.com/uoregon-libraries/newspaper-curation-app/src/duration"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

type count struct {
	Title  string
	Issues int
	Pages  int
}

// aggregation is our batch-generation-view of MOC issue aggregation data - we
// get transform the raw for easy display, but also pull the full list of
// "ready for batching" issues in order to show detailed information about
// embargoes, how long the longest issues have been waiting, etc.
type aggregation struct {
	MOC              *models.MOC
	Counts           []count
	ReadyForBatching *issuequeue.Queue
	Age              string
}

func (a *aggregation) appendCount(agg *models.IssueAggregation, title string, steps ...schema.WorkflowStep) {
	var issues, pages int
	for _, step := range steps {
		issues += int(agg.Counts[step].IssueCount)
		pages += int(agg.Counts[step].TotalPages)
	}

	if issues > 0 {
		a.Counts = append(a.Counts, count{Title: title, Issues: issues, Pages: pages})
	}
}

// getAggregations builds our template-friendly structures, transforming the
// data so it helps people decide what to batch, and removing data that would
// just be noise, such as MOCs which have no issues ready for batching.
func getAggregations(aggs []*models.IssueAggregation) ([]*aggregation, error) {
	var list []*aggregation
	for _, agg := range aggs {
		if agg.Counts[schema.WSReadyForBatching].IssueCount == 0 {
			continue
		}

		// Get "ready for batching" issues fully loaded so we can provide embargo /
		// stale details before other counts
		var a = &aggregation{MOC: agg.MOC}
		var issues, err = models.Issues().MOC(a.MOC.Code).InWorkflowStep(schema.WSReadyForBatching).BatchID(0).Fetch()
		if err != nil {
			return nil, fmt.Errorf("fetching issues for %q: %w", a.MOC.Code, err)
		}

		var tempQ = issuequeue.New()
		for _, issue := range issues {
			err = tempQ.Append(issue)
			if err != nil {
				return nil, fmt.Errorf("appending issue %q to aggregation queue %q: %w", issue.Key(), a.MOC.Code, err)
			}
		}
		var embargoed = func(issue *issuequeue.Issue) bool { return issue.Embargoed }
		var notEmbargoed = func(issue *issuequeue.Issue) bool { return !issue.Embargoed }
		var embargoedQ = tempQ.Filter(embargoed)
		a.ReadyForBatching = tempQ.Filter(notEmbargoed)

		// This is theoretically possible if all "ready" issues are in fact still
		// under embargo. The information may still be worth displaying, but it
		// can't be acted upon, so we probably want to make a separate view if
		// people are just looking for high-level stats.
		if a.ReadyForBatching.Len() == 0 {
			continue
		}

		var d = duration.FromDays(int(a.ReadyForBatching.DaysStale))
		if d.Zero() {
			a.Age = "Less than a day"
		} else {
			a.Age = d.String()
		}

		if embargoedQ.Len() > 0 {
			a.Counts = append(
				a.Counts,
				count{Title: "Ready, but under embargo", Issues: embargoedQ.Len(), Pages: embargoedQ.Pages},
			)
		}

		// Add counts for other useful data points, merging similar ones to reduce noise
		a.appendCount(agg, "Uploaded, not in NCA", schema.WSSFTP, schema.WSScan)
		a.appendCount(agg, "Waiting on user action", schema.WSAwaitingPageReview, schema.WSReadyForMetadataEntry, schema.WSAwaitingMetadataReview)
		a.appendCount(agg, "In NCA, processing", schema.WSAwaitingProcessing, schema.WSReadyForMETSXML)
		a.appendCount(agg, "In NCA, flagged (unfixable errors)", schema.WSUnfixableMetadataError)

		list = append(list, a)
	}

	return list, nil
}

// Q wraps issuequeue.Queue to add some context for the template to display. Q
// is not, however, likely to be found anywhere near farpoint. At least not in
// the NCA continuum.
type Q struct {
	Sequence int
	MOC      *models.MOC
	Queue    *issuequeue.Queue
}
