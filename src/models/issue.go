package models

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// Workflow steps in-process issues may have
var allowedWorkflowSteps = []schema.WorkflowStep{
	schema.WSUnfixableMetadataError,
	schema.WSAwaitingProcessing,
	schema.WSAwaitingPageReview,
	schema.WSReadyForMetadataEntry,
	schema.WSAwaitingMetadataReview,
	schema.WSReadyForMETSXML,
	schema.WSReadyForBatching,
	schema.WSReadyForRebatching,
	schema.WSInProduction,
}

// Issue contains metadata about an issue for the various workflow tools' use
type Issue struct {
	ID int64 `sql:",primary"`

	// Metadata
	MARCOrgCode   string
	LCCN          string
	Date          string
	DateAsLabeled string
	Volume        string

	// Titles are tied to issues by LCCN, and all issues must have a title in the
	// database, so we load these when we load the issue data
	Title *Title `sql:"-"`

	// This field is a bit confusing, but it is the NDNP field for the issue
	// "number", which is actually a string since it can contain things like
	// "ISSUE XIX"
	Issue         string
	Edition       int
	EditionLabel  string
	PageLabelsCSV string
	PageLabels    []string `sql:"-"`

	// PageCount is a cached value in the database for quickly getting the length
	// of page labels. As such, it will be meaningless until curation, and could
	// still be incorrect prior to metadata review.
	PageCount int

	/* Workflow information to keep track of the issue and what it needs */

	BatchID                int64               // Which batch (if any) is this issue a part of?
	Location               string              // Where is this issue on disk?
	BackupLocation         string              // Where is the original backup located?  (born-digital only)
	HumanName              string              // What is the issue's "human" name (for consistent folder naming)?
	IsFromScanner          bool                // Is the issue scanned in-house?  (Born-digital == false)
	WorkflowStepString     string              `sql:"workflow_step"` // If set, tells us what "human workflow" step we're on
	WorkflowStep           schema.WorkflowStep `sql:"-"`
	WorkflowOwnerID        int64               // Whose "desk" is this currently on?
	WorkflowOwnerExpiresAt time.Time           // When does the workflow owner lose ownership?
	MetadataEntryUserID    int64               // Who entered metadata?
	MetadataEnteredAt      time.Time           // When was metadata last saved for this issue?
	ReviewedByUserID       int64               // Who reviewed metadata last?
	MetadataApprovedAt     time.Time           // When was metadata approved / how long has this been waiting to batch?
	RejectedByUserID       int64               // If not approved, who rejected the metadata?
	Ignored                bool                // Is the issue bad / in prod / otherwise skipped from workflow scans?
	DraftComment           string              // Any comment the curator is passing on to the reviewer

	// actions holds the lazy-loaded list of actions tied to an issue, ordered
	// by the most recent to the oldest
	actions []*Action

	// lazy-loaded list of jobs tied to this issue, directly or indirectly via a pipeline
	jobs []*Job

	// lazy-loaded list of pipelines directly tied to this issue
	pipelines []*Pipeline
}

// FlaggedIssue is a record indicating an issue which was flagged for removal
// from a batch
type FlaggedIssue struct {
	Issue  *Issue
	User   *User
	When   time.Time
	Reason string
}

func findFlaggedIssues(where string, args ...any) ([]*FlaggedIssue, error) {
	type _row struct {
		IssueID         int64
		FlaggedByUserID int64
		Reason          string
		CreatedAt       time.Time
	}
	var rows []*_row

	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.Select("batches_flagged_issues", &_row{}).Where(where, args...).AllObjects(&rows)

	var issues = make([]*FlaggedIssue, len(rows))
	for i, row := range rows {
		var issue, err = FindIssue(row.IssueID)
		if err != nil {
			return nil, fmt.Errorf("findFlaggedIssues(): error querying issue %d: %w", row.IssueID, err)
		}
		if issue == nil {
			return nil, fmt.Errorf("findFlaggedIssues(): error querying issue %d: no issue found", row.IssueID)
		}

		var user = FindUserByID(row.FlaggedByUserID)
		if user == EmptyUser {
			return nil, fmt.Errorf("findFlaggedIssues(): unable to find user %d", row.FlaggedByUserID)
		}

		issues[i] = &FlaggedIssue{
			Issue:  issue,
			User:   user,
			Reason: row.Reason,
			When:   row.CreatedAt,
		}
	}

	return issues, nil
}

// FindFlaggedIssue returns a flagged issue identified by the issue's batch id and id
func FindFlaggedIssue(batchID, issueID int64) (*FlaggedIssue, error) {
	var list, err = findFlaggedIssues("batch_id = ? AND issue_id = ?", batchID, issueID)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, sql.ErrNoRows
	}
	return list[0], err
}

// CreateIssueFromUpload is a single-purpose issue-builder just for storing an
// issue which was queued via our uploads controller
func CreateIssueFromUpload(moc, lccn, rawDate string, edition int, loc string, scanned bool) (*Issue, error) {
	var issue = &Issue{
		MARCOrgCode:   moc,
		LCCN:          lccn,
		Date:          rawDate,
		Edition:       edition,
		Location:      loc,
		IsFromScanner: scanned,
		WorkflowStep:  schema.WSAwaitingProcessing,
	}
	return issue, issue.Save(ActionTypeInternalProcess, SystemUser.ID, "Issue data initialized in NCA")
}

// FindIssue looks for an issue by its id
func FindIssue(id int64) (*Issue, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	var i = &Issue{}
	var ok = op.Select("issues", &Issue{}).Where("id = ?", id).First(i)
	if !ok {
		return nil, op.Err()
	}
	var err error
	i.Title, err = FindTitleByLCCN(i.LCCN)
	if err != nil {
		return nil, err
	}
	i.deserialize()
	return i, op.Err()
}

// FindIssuesByKey looks for all issues in the database that have the given
// issue key.  Having more than one is an error, but we allow users to save
// metadata in "draft" form, so we have to be able to test for dupes later in
// the process
func FindIssuesByKey(key string) ([]*Issue, error) {
	var parts = strings.Split(key, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid issue key %q", key)
	}
	if len(parts[1]) != 10 {
		return nil, fmt.Errorf("invalid issue key %q", key)
	}

	var lccn = parts[0]
	var dateShort = parts[1][:8]
	var date = fmt.Sprintf("%s-%s-%s", dateShort[:4], dateShort[4:6], dateShort[6:8])

	var ed, err = strconv.Atoi(parts[1][8:])
	if err != nil {
		return nil, fmt.Errorf("invalid issue key %q", key)
	}

	return Issues().LCCN(lccn).date(date).edition(ed).Fetch()
}

// FindIssueByKey returns the first issue with the given key
func FindIssueByKey(key string) (*Issue, error) {
	var list, err = FindIssuesByKey(key)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, nil
	}
	return list[0], nil
}

// FindInProcessIssues returns all issues which have been entered in the
// workflow system, but haven't yet gone through all the way to the point of
// being batched and approved for production
func FindInProcessIssues() ([]*Issue, error) {
	// This seems really awful, but every issue that's in the system and isn't
	// ignored is in process today. We keep this helper around because at least
	// the function name says something clearly compared to the code below.
	return Issues().Fetch()
}

// Key returns the standardized issue key for this DB issue
func (i *Issue) Key() string {
	return schema.IssueKey(i.LCCN, i.Date, i.Edition)
}

// DateEdition returns the date+edition string used by our general schema
func (i *Issue) DateEdition() string {
	return schema.IssueDateEdition(i.Date, i.Edition)
}

// METSFile returns the canonical path to an issue's METS file. This will only
// have meaning for issues still on a filesystem somewhere.
func (i *Issue) METSFile() string {
	return filepath.Join(i.Location, i.DateEdition()+".xml")
}

// BuildJob returns a new Job instance for manipulating this issue in some way
func (i *Issue) BuildJob(t JobType, args map[string]string) *Job {
	var j = NewJob(t, args)
	j.ObjectID = i.ID
	j.ObjectType = JobObjectTypeIssue
	return j
}

// AllWorkflowActions loads all actions tied to this issue and orders them in
// chronological order (the newest are at the end of the list)
func (i *Issue) AllWorkflowActions() []*Action {
	if i.actions == nil {
		// Yup, we deliberately ignore errors here.  Bah.
		i.actions, _ = FindActionsForIssue(i.ID)
	}

	return i.actions
}

// WorkflowActions loads meaningful (to curators and reviewers) actions tied to
// this issue and orders them in chronological order (the newest are at the end
// of the list)
func (i *Issue) WorkflowActions() []*Action {
	var actions = i.AllWorkflowActions()
	var meaningful []*Action
	for _, a := range actions {
		if a.important() {
			meaningful = append(meaningful, a)
		}
	}

	return meaningful
}

// Claim sets the workflow owner to the given user id, and sets the expiration
// time to a week from now
func (i *Issue) Claim(byUserID int64) error {
	i.claim(byUserID)
	return i.Save(ActionTypeClaim, byUserID, "")
}

// Jobs returns a list of jobs associated with this issue via its
// issue-specific pipelines. The result is cached and must not be reused if
// it's critical to read the latest jobs from the database.
func (i *Issue) Jobs() ([]*Job, error) {
	if i.jobs != nil {
		return i.jobs, nil
	}

	var pList, err = i.Pipelines()
	if err != nil {
		return nil, err
	}

	// We gather jobs before putting them in i.jobs to keep i.jobs == nil until
	// we're sure everything succeeded
	var allJobs []*Job
	for _, p := range pList {
		var jList, err = p.Jobs()
		if err != nil {
			return nil, err
		}
		allJobs = append(allJobs, jList...)
	}

	i.jobs = allJobs
	return i.jobs, nil
}

// Pipelines returns a list of pipelines associated with this issue. The result
// is cached and must not be reused if it's critical to read the latest
// pipelines from the database.
func (i *Issue) Pipelines() ([]*Pipeline, error) {
	var err error
	if i.pipelines == nil {
		i.pipelines, err = findPipelines("object_type = ? AND object_id = ?", JobObjectTypeIssue, i.ID)
	}

	return i.pipelines, err
}

// claim updates metadata without writing to the database so internal
// functions can use this as just one step of the update process
func (i *Issue) claim(byUserID int64) {
	// *Never* let an empty or system user claim anything!
	if byUserID <= 0 {
		return
	}

	i.WorkflowOwnerID = byUserID
	i.WorkflowOwnerExpiresAt = time.Now().Add(time.Hour * 24 * 7)
}

// Unclaim removes the workflow owner and resets the workflow expiration time
func (i *Issue) Unclaim(byUserID int64) error {
	i.unclaim()
	return i.Save(ActionTypeUnclaim, byUserID, "")
}

// unclaim updates metadata without writing to the database so internal
// functions can use this as just one step of the update process
func (i *Issue) unclaim() {
	i.WorkflowOwnerID = 0
	i.WorkflowOwnerExpiresAt = time.Time{}
}

// SetCurated records the curator, changes the owner of the issue if it was
// previously rejected in metadata review, and stores the draft comment, if
// any was present.
func (i *Issue) SetCurated(curatorID int64) error {
	// Update workflow step and record the curator id
	i.MetadataEntryUserID = curatorID
	i.MetadataEnteredAt = time.Now()
	i.unclaim()

	// If this was previously rejected, put it back on the reviewer's desk
	if i.RejectedByUserID != 0 {
		i.claim(i.RejectedByUserID)
	}

	var message = i.DraftComment
	i.DraftComment = ""
	return i.Save(ActionTypeMetadataEntry, curatorID, message)
}

// ApproveMetadata moves the issue to the final workflow step (e.g., no more
// manual steps) and sets the reviewer id to that which was passed in
func (i *Issue) ApproveMetadata(reviewerID int64) error {
	i.unclaim()
	i.MetadataApprovedAt = time.Now()
	i.ReviewedByUserID = reviewerID
	i.WorkflowStep = schema.WSReadyForMETSXML
	return i.Save(ActionTypeMetadataApproval, reviewerID, "")
}

// RejectMetadata sends the issue back to the metadata entry user and saves the
// reviewer's notes
func (i *Issue) RejectMetadata(reviewerID int64, notes string) error {
	i.claim(i.MetadataEntryUserID)
	i.RejectedByUserID = reviewerID
	i.WorkflowStep = schema.WSReadyForMetadataEntry
	return i.Save(ActionTypeMetadataRejection, reviewerID, notes)
}

// ReportError adds an error message to the issue and flags it as being in the
// "unfixable" state.  That state basically says that nobody can use NCA to fix
// the problem, and it needs to be pulled and processed by hand.
func (i *Issue) ReportError(userID int64, message string) error {
	i.WorkflowStep = schema.WSUnfixableMetadataError
	i.unclaim()
	return i.Save(ActionTypeReportUnfixableError, userID, message)
}

// returnFor implements the issue and action logic we want when returning an
// errored issue to NCA.  If deskID is nonzero, the issue is forced to the
// given user's desk.
func (i *Issue) returnFor(ws schema.WorkflowStep, ac ActionType, managerID, workflowOwnerID int64, msg string) error {
	if i.WorkflowStep != schema.WSUnfixableMetadataError {
		return fmt.Errorf("invalid WorkflowStep %q: issue must be unfixable", i.WorkflowStep)
	}
	i.unclaim()
	i.WorkflowStep = ws
	if workflowOwnerID > 0 {
		i.claim(workflowOwnerID)
	}
	return i.Save(ac, managerID, msg)
}

// ReturnForCuration is a manager-only action which forces an issue back to the
// metadata entry queue after it had been marked unfixable.  If workflowOwnerID
// is nonzero, that user becomes the new owner of the issue.
func (i *Issue) ReturnForCuration(managerID, workflowOwnerID int64, comment string) error {
	return i.returnFor(schema.WSReadyForMetadataEntry, ActionTypeReturnCurate, managerID, workflowOwnerID, comment)
}

// ReturnForReview is a manager-only action which forces an issue back to the
// metadata review queue after it had been marked unfixable.  If
// workflowOwnerID is nonzero, that user becomes the new owner of the issue.
func (i *Issue) ReturnForReview(managerID, workflowOwnerID int64, comment string) error {
	return i.returnFor(schema.WSAwaitingMetadataReview, ActionTypeReturnReview, managerID, workflowOwnerID, comment)
}

// PrepForRemoval sets up the issue's metadata such that nothing else will
// try to process it in any way as it waits for a job (or even a manual action)
// to remove it
func (i *Issue) PrepForRemoval(managerID int64, message string) error {
	i.unclaim()
	i.WorkflowStep = schema.WSAwaitingProcessing
	return i.Save(ActionTypeRemoveErrorIssue, managerID, message)
}

// Save creates or updates the issue with an associated action and optional message
func (i *Issue) Save(action ActionType, userID int64, message string) error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	op.BeginTransaction()
	defer op.EndTransaction()
	return i.SaveOp(op, action, userID, message)
}

// SaveWithoutAction creates or updates the Issue in the issues table without
// associating any kind of action.  This should be used sparingly, however, as
// the action log is key to debugging a variety of issues as well as
// determining what's going on with an issue.
func (i *Issue) SaveWithoutAction() error {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug
	return i.SaveOpWithoutAction(op)
}

// SaveOp creates or updates the Issue in the issues table with a custom operation
func (i *Issue) SaveOp(op *magicsql.Operation, action ActionType, userID int64, message string) error {
	var a = NewIssueAction(i.ID, action)
	a.UserID = userID
	a.Message = message
	i.actions = append(i.actions, a)

	_ = a.SaveOp(op)
	_ = i.SaveOpWithoutAction(op)
	return op.Err()
}

// SaveOpWithoutAction is the transaction-friendly SaveWithoutAction. This
// should of course be used sparingly.
func (i *Issue) SaveOpWithoutAction(op *magicsql.Operation) error {
	var valid bool
	for _, validWS := range allowedWorkflowSteps {
		if string(i.WorkflowStep) == string(validWS) {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("issue doesn't have a valid workflow step, %q", i.WorkflowStep)
	}

	i.serialize()
	op.Save("issues", i)
	i.setHumanName()
	return op.Err()
}

// serialize prepares struct data to work with the database fields better
func (i *Issue) serialize() {
	i.PageLabelsCSV = strings.Join(i.PageLabels, "␟")
	i.WorkflowStepString = string(i.WorkflowStep)
	i.PageCount = len(i.PageLabels)
}

// deserialize performs operations necessary to get the database data into a more
// useful Go structure
func (i *Issue) deserialize() {
	i.PageLabels = strings.Split(i.PageLabelsCSV, "␟")
	i.WorkflowStep = schema.WorkflowStep(i.WorkflowStepString)
	i.setHumanName()
}

// setHumanName ensures the human name is set up, but only if it's blank and
// has a DB id
func (i *Issue) setHumanName() {
	if i.HumanName == "" && i.ID != 0 {
		var dte = schema.IssueDateEdition(i.Date, i.Edition)
		i.HumanName = fmt.Sprintf("%s-%s-%d", i.LCCN, dte, i.ID)
	}
}

// deserializeIssues runs deserialize() against all issues in the list
func deserializeIssues(list []*Issue) {
	// Pull all titles so we hit the titles table only once.  We can ignore the
	// error here because it'll be stored on the operation and returned from
	// whatever called this.
	var titles, _ = Titles()
	for _, i := range list {
		i.Title = titles.FindByLCCN(i.LCCN)
		i.deserialize()
	}
}

// SchemaIssue returns an extremely over-simplified representation of this
// issue as a schema.Issue instance for ensuring consistent representation of
// things like issue keys
func (i *Issue) SchemaIssue() (*schema.Issue, error) {
	var si = &schema.Issue{
		DatabaseID:   i.ID,
		RawDate:      i.Date,
		Edition:      i.Edition,
		Title:        i.Title.SchemaTitle(),
		Location:     i.Location,
		MARCOrgCode:  i.MARCOrgCode,
		WorkflowStep: schema.WorkflowStep(i.WorkflowStep),
	}

	// Bad metadata can happen when saving an issue without validation, either
	// via autosaves that happen when looking through pages, or saving a draft.
	// Therefore, on bad dates, we return an error, but also a semi-usable issue
	// for situations where we don't care if the metadata isn't quite right.
	var _, err = time.Parse("2006-01-02", i.Date)
	if err != nil {
		err = fmt.Errorf("invalid time format (%s) in database issue", i.Date)
	}

	return si, err
}

// FindCompletedIssuesReadyForRemoval returns all issues which are complete and
// no longer needed in our workflow: tied to a closed (live_done) batch and
// ignored by NCA, but still contain a location
func FindCompletedIssuesReadyForRemoval() ([]*Issue, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug

	var list []*Issue
	var cond = "batch_id IN (SELECT id FROM batches WHERE status = ?) AND ignored = 1 AND location <> ''"
	op.Select("issues", &Issue{}).Where(cond, BatchStatusLiveDone).AllObjects(&list)
	deserializeIssues(list)
	return list, op.Err()
}

// FindIssuesLackingMetadataEntryDate is a one-off to help migrate legacy
// issues. It's dumb and shouldn't live here. Blah.
func FindIssuesLackingMetadataEntryDate() ([]*Issue, error) {
	var op = dbi.DB.Operation()
	op.Dbg = dbi.Debug

	var list []*Issue
	op.Select("issues", &Issue{}).Where("metadata_entered_at IS NULL").AllObjects(&list)
	deserializeIssues(list)
	return list, op.Err()
}
