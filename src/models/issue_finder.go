package models

import (
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// IssueFinder is a pseudo-DSL for easily creating queries without needing to
// know the underlying table structure
type IssueFinder struct {
	*coreFinder
}

// Issues returns an IssueFinder: a scoped object for simple filtering of the
// issues table with a very narrow DSL
func Issues() *IssueFinder {
	var f = newCoreFinder("issues", &Issue{})
	f.conditions["ignored = ?"] = false
	return &IssueFinder{coreFinder: f}
}

// LCCN returns a scope for finding issues with a particular title
func (f *IssueFinder) LCCN(lccn string) *IssueFinder {
	f.conditions["lccn = ?"] = lccn
	return f
}

// MOC returns a scope for finding issues with a particular awardee (MARC Org Code)
func (f *IssueFinder) MOC(moc string) *IssueFinder {
	f.conditions["marc_org_code = ?"] = moc
	return f
}

func (f *IssueFinder) date(date string) *IssueFinder {
	f.conditions["date = ?"] = date
	return f
}
func (f *IssueFinder) edition(ed int) *IssueFinder {
	f.conditions["edition = ?"] = ed
	return f
}

// InWorkflowStep filters issues by a given workflow step. Most common use:
//
//   - WSAwaitingProcessing: issues which are "invisible" to the UI because
//     some automated process needs to run or is running
//   - WSAwaitingPageReview: issues we have to regularly check to see if page
//     review renaming is complete
//   - WSReadyForBatching: issues with no batch id in this step are gathered up
//     for generating a new ONI batch
func (f *IssueFinder) InWorkflowStep(ws schema.WorkflowStep) *IssueFinder {
	f.conditions["workflow_step = ?"] = string(ws)
	return f
}

// BatchID filters issues associated with the given batch id
func (f *IssueFinder) BatchID(batchID int64) *IssueFinder {
	f.conditions["batch_id = ?"] = batchID
	return f
}

// AllowIgnored removes the standard "ignored = false" clause. This is useful
// for some very specific cases, like showing issues belonging to live batches.
func (f *IssueFinder) AllowIgnored() *IssueFinder {
	delete(f.conditions, "ignored = ?")
	return f
}

// OnDesk filters issues "owned" by a given user id
func (f *IssueFinder) OnDesk(userID int64) *IssueFinder {
	f.conditions["workflow_owner_id = ?"] = userID
	f.conditions["workflow_owner_expires_at IS NOT NULL"] = nil
	f.conditions["workflow_owner_expires_at > ?"] = time.Now()
	return f
}

// Available filters issues to just those which are "available". We
// define "available" as any issue without an owner or where ownership expired
// (e.g., an issue was sitting on somebody's desk for several days).
func (f *IssueFinder) Available() *IssueFinder {
	f.conditions["workflow_owner_id = 0 OR workflow_owner_expires_at < ?"] = time.Now()
	return f
}

// NotCuratedBy adds a filter for ensuring the given user didn't curate an
// issue, for preventing us from loading hundreds of issues that cannot in fact
// be reviewed by a given logged-in user.
func (f *IssueFinder) NotCuratedBy(userID int64) *IssueFinder {
	f.conditions["metadata_entry_user_id <> ?"] = userID
	return f
}

// Limit sets the max issues to return
func (f *IssueFinder) Limit(limit int) *IssueFinder {
	f.lim = limit
	return f
}

// OrderBy sets an order for this finder.
//
// TODO: This currently requires a raw SQL order string which ties business
// logic and DB schema too tightly. Not sure the best way to address this.
func (f *IssueFinder) OrderBy(order string) *IssueFinder {
	f.ord = order
	return f
}

// Fetch returns all issues this scoped finder represents
func (f *IssueFinder) Fetch() ([]*Issue, error) {
	var list []*Issue
	var err = f.coreFinder.Fetch(&list)
	if err == nil {
		deserializeIssues(list)
	}
	return list, err
}

// Count returns the number of records this query would return
func (f *IssueFinder) Count() (uint64, error) {
	return f.coreFinder.Count()
}
