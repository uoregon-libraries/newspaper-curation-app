package issuequeue

import (
	"fmt"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// Issue wraps a db Issue to compute staleness and embargo data
type Issue struct {
	*models.Issue
	DaysStale float64
	Embargoed bool
}

func wrapIssue(dbIssue *models.Issue) (*Issue, error) {
	if dbIssue.Title == nil {
		return nil, fmt.Errorf("wrapping issue: invalid: no associated title")
	}

	var issueDate, err = time.Parse("2006-01-02", dbIssue.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid date %q: %w", dbIssue.Date, err)
	}

	var i = &Issue{Issue: dbIssue}

	var embargoLiftDate time.Time
	embargoLiftDate, err = i.Title.CalculateEmbargoLiftDate(issueDate)
	if err != nil {
		return nil, fmt.Errorf("parsing title's embargo duration: %w", err)
	}

	if embargoLiftDate.After(time.Now()) {
		i.Embargoed = true
	}

	// Embargoed issues can be waiting for a while before their embargo is
	// lifted, so we have to consider them stale based on the newer date:
	// metadata approval or embargo lifting.
	if i.MetadataApprovedAt.Before(embargoLiftDate) {
		i.DaysStale = time.Since(embargoLiftDate).Hours() / 24.0
	} else {
		i.DaysStale = time.Since(i.MetadataApprovedAt).Hours() / 24.0
	}

	// If the title isn't validated yet, its issues can't be queued
	if !i.Title.ValidLCCN {
		return nil, fmt.Errorf("LCCN %q hasn't been validated", i.LCCN)
	}

	return i, nil
}
