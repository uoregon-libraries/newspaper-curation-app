package batchmakerhandler

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

type count struct {
	Title  string
	Issues int64
	Pages  int64
}

// aggregation is our batch-generation-view version of the MOC issue aggregation data
type aggregation struct {
	MOC    *models.MOC
	Counts []count
}

func (a *aggregation) appendCount(agg *models.IssueAggregation, title string, steps ...schema.WorkflowStep) {
	var issues, pages int64
	for _, step := range steps {
		issues += agg.Counts[step].IssueCount
		pages += agg.Counts[step].TotalPages
	}

	if issues > 0 {
		a.Counts = append(a.Counts, count{Title: title, Issues: issues, Pages: pages})
	}
}

// getAggregations builds our template-friendly structures, transforming
// the data so it helps people decide what to batch
func getAggregations(aggs []*models.IssueAggregation) []*aggregation {
	var list []*aggregation
	for _, agg := range aggs {
		if agg.Counts[schema.WSReadyForBatching].IssueCount == 0 {
			continue
		}

		var a = &aggregation{MOC: agg.MOC}

		// Add counts for useful data points, merging similar ones to reduce noise.
		//
		// NOTE: order matters here! We want users to see the "Ready for batching"
		// numbers first, and then prioritize items that are "further back" in the
		// process.
		a.appendCount(agg, "Ready for batching", schema.WSReadyForBatching)
		a.appendCount(agg, "Uploaded, not in NCA", schema.WSSFTP, schema.WSScan)
		a.appendCount(agg, "Waiting on user action", schema.WSAwaitingPageReview, schema.WSReadyForMetadataEntry, schema.WSAwaitingMetadataReview)
		a.appendCount(agg, "In NCA, processing", schema.WSAwaitingProcessing, schema.WSReadyForMETSXML)
		a.appendCount(agg, "In NCA, flagged (unfixable errors)", schema.WSUnfixableMetadataError)

		list = append(list, a)
	}

	return list
}
