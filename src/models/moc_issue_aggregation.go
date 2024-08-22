package models

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// rawMOCIssueAggregation is our data model for the moc_issue_aggregation view.
// This is a data-only model, and part of an ongoing effort to separate data
// structures from business-logic structures.
type rawMOCIssueAggregation struct {
	ID           int64 `sql:",primary"`
	Code         string
	Name         string
	WorkflowStep string
	IssueCount   int64
	TotalPages   int64
}

// rawMOCIssueAggregations returns the raw list of aggregated MOC data
func rawMOCIssueAggregations() ([]*rawMOCIssueAggregation, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var list []*rawMOCIssueAggregation
	op.Select("moc_issue_aggregation", &rawMOCIssueAggregation{}).AllObjects(&list)
	return list, op.Err()
}

// WorkflowCounts just holds the number of issues and pages aggregated for a
// given MOC and workflow step
type WorkflowCounts struct {
	IssueCount int64
	TotalPages int64
}

// IssueAggregation holds data representing a single organization's aggregate
// issue and page counts, broken up by workflow step
type IssueAggregation struct {
	MOC    *MOC
	Counts map[schema.WorkflowStep]WorkflowCounts
}

func newIssueAggregation(raw *rawMOCIssueAggregation) *IssueAggregation {
	return &IssueAggregation{
		MOC: &MOC{
			ID:   raw.ID,
			Code: raw.Code,
			Name: raw.Name,
		},
		Counts: make(map[schema.WorkflowStep]WorkflowCounts),
	}
}

// MOCIssueAggregations returns a list of aggregated data, one per Marc Org
// Code, telling us a high-level view about the issues related to the given
// MOC. No data is returned if the MOC has no associated issues.
func MOCIssueAggregations() ([]*IssueAggregation, error) {
	var rawList, err = rawMOCIssueAggregations()
	if err != nil {
		return nil, fmt.Errorf("reading raw MOC issue aggregations: %w", err)
	}

	var list []*IssueAggregation
	var byID = make(map[int64]*IssueAggregation)
	for _, rawAgg := range rawList {
		var agg, ok = byID[rawAgg.ID]
		if !ok {
			agg = newIssueAggregation(rawAgg)
			byID[rawAgg.ID] = agg
			list = append(list, agg)
		}

		var ws = schema.WorkflowStep(rawAgg.WorkflowStep)
		agg.Counts[ws] = WorkflowCounts{
			IssueCount: rawAgg.IssueCount,
			TotalPages: rawAgg.TotalPages,
		}
	}

	return list, nil
}
