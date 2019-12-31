package db

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Nerdmaster/magicsql"
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
	schema.WSInProduction,
}

// Issue contains metadata about an issue for the various workflow tools' use
type Issue struct {
	ID int `sql:",primary"`

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

	/* Workflow information to keep track of the issue and what it needs */

	BatchID                int                 // Which batch (if any) is this issue a part of?
	Error                  string              // If set, a metadata curator reported a problem
	Location               string              // Where is this issue on disk?
	MasterBackupLocation   string              // Where is the master backup located?  (born-digital only)
	HumanName              string              // What is the issue's "human" name (for consistent folder naming)?
	IsFromScanner          bool                // Is the issue scanned in-house?  (Born-digital == false)
	WorkflowStepString     string              `sql:"workflow_step"` // If set, tells us what "human workflow" step we're on
	WorkflowStep           schema.WorkflowStep `sql:"-"`
	WorkflowOwnerID        int                 // Whose "desk" is this currently on?
	WorkflowOwnerExpiresAt time.Time           // When does the workflow owner lose ownership?
	MetadataEntryUserID    int                 // Who entered metadata?
	ReviewedByUserID       int                 // Who reviewed metadata?
	MetadataApprovedAt     time.Time           // When was metadata approved / how long has this been waiting to batch?
	RejectionNotes         string              // If rejected (during metadata review), this tells us why
	RejectedByUserID       int                 // Who did the rejection?
	Ignored                bool                // Is the issue bad / in prod / otherwise skipped from workflow scans?
}

// NewIssue creates an issue ready for saving to the issues table
func NewIssue(moc, lccn, dt string, ed int) *Issue {
	return &Issue{MARCOrgCode: moc, LCCN: lccn, Date: dt, Edition: ed, WorkflowStep: schema.WSAwaitingProcessing}
}

// findIssues is a centralized finder that auto-skips issues flagged as ignored
// and auto-deserializes the issues returned
func findIssues(cond string, args ...interface{}) ([]*Issue, error) {
	var op = DB.Operation()
	op.Dbg = Debug

	var list []*Issue
	if cond == "" {
		cond = "ignored = ?"
	} else {
		cond = "(" + cond + ")" + " AND ignored = ?"
	}
	args = append(args, false)
	op.Select("issues", &Issue{}).Where(cond, args...).AllObjects(&list)
	deserializeIssues(list)
	return list, op.Err()
}

// FindIssue looks for an issue by its id
func FindIssue(id int) (*Issue, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var i = &Issue{}
	var ok = op.Select("issues", &Issue{}).Where("id = ?", id).First(i)
	if !ok {
		return nil, op.Err()
	}
	var err error
	i.Title, err = FindTitle("lccn = ?", i.LCCN)
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

	return findIssues("lccn = ? AND date = ? AND edition = ?", lccn, date, ed)
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

// FindIssueByLocation returns the first issue with the given location
func FindIssueByLocation(location string) (*Issue, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var i *Issue
	var list, err = findIssues("location = ?", location)
	if len(list) != 0 {
		i = list[0]
	}
	return i, err
}

// FindInProcessIssues returns all issues which have been entered in the
// workflow system, but haven't yet gone through all the way to the point of
// being batched and approved for production
func FindInProcessIssues() ([]*Issue, error) {
	return findIssues("")
}

// FindIssuesAwaitingProcessing returns all issues which should be considered
// "invisible" to the UI - these are untouchable until some automated process
// is complete
func FindIssuesAwaitingProcessing() ([]*Issue, error) {
	return findIssues("workflow_step = ?", string(schema.WSAwaitingProcessing))
}

// FindIssuesByBatchID returns all issues associated with the given batch id
func FindIssuesByBatchID(batchID int) ([]*Issue, error) {
	return findIssues("batch_id = ?", batchID)
}

// FindIssuesOnDesk returns all issues "owned" by a given user id
func FindIssuesOnDesk(userID int) ([]*Issue, error) {
	return findIssues(`
		workflow_owner_id = ? AND
		workflow_owner_expires_at IS NOT NULL AND
		workflow_owner_expires_at > ?`, userID, time.Now())
}

// FindIssuesInPageReview looks for all issues currently awaiting page review
// and returns them
func FindIssuesInPageReview() ([]*Issue, error) {
	return findIssues("workflow_step = ?", string(schema.WSAwaitingPageReview))
}

// FindIssuesReadyForBatching looks for all issues which are in the
// WSReadyForBatching workflow step and have no batch ID
func FindIssuesReadyForBatching() ([]*Issue, error) {
	return findIssues("workflow_step = ? AND batch_id = 0", string(schema.WSReadyForBatching))
}

// FindAvailableIssuesByWorkflowStep looks for all "available" issues with the
// requested workflow step and returns them.  We define "available" as:
//
// - No owner (or owner expired)
// - Have not been reported as having errors
func FindAvailableIssuesByWorkflowStep(ws schema.WorkflowStep) ([]*Issue, error) {
	return findIssues("workflow_step = ? AND (workflow_owner_id = 0 OR workflow_owner_expires_at < ?) AND error = ''",
		string(ws), time.Now().Format("2006-01-02 15:04:05"))
}

// FindIssuesWithErrors returns all issues with an error reported by metadata
// entry personnel
func FindIssuesWithErrors() ([]*Issue, error) {
	return findIssues("workflow_step = ?", string(schema.WSUnfixableMetadataError))
}

// Key returns the standardized issue key for this DB issue
func (i *Issue) Key() string {
	return schema.IssueKey(i.LCCN, i.Date, i.Edition)
}

// DateEdition returns the date+edition string used by our general schema
func (i *Issue) DateEdition() string {
	return schema.IssueDateEdition(i.Date, i.Edition)
}

// Claim sets the workflow owner to the given user id, and sets the expiration
// time to a week from now
func (i *Issue) Claim(byUserID int) {
	i.WorkflowOwnerID = byUserID
	i.WorkflowOwnerExpiresAt = time.Now().Add(time.Hour * 24 * 7)
}

// Unclaim removes the workflow owner and resets the workflow expiration time
func (i *Issue) Unclaim() {
	i.WorkflowOwnerID = 0
	i.WorkflowOwnerExpiresAt = time.Time{}
}

// ApproveMetadata moves the issue to the final workflow step (e.g., no more
// manual steps) and sets the reviewer id to that which was passed in
func (i *Issue) ApproveMetadata(reviewerID int) {
	i.Unclaim()
	i.MetadataApprovedAt = time.Now()
	i.ReviewedByUserID = reviewerID
	i.WorkflowStep = schema.WSReadyForMETSXML
}

// RejectMetadata sends the issue back to the metadata entry user and saves the
// reviewer's notes
//
// TODO: if we ever display rejection user, bear in mind that 0 means it's
// rejected by a system process rather than a person
func (i *Issue) RejectMetadata(reviewerID int, notes string) {
	i.Claim(i.MetadataEntryUserID)
	i.WorkflowStep = schema.WSReadyForMetadataEntry
	i.RejectionNotes = notes
	i.RejectedByUserID = reviewerID
}

// ReportError adds an error message to the issue and flags it as being in the
// "unfixable" state.  That state basically says that nobody can use NCA to fix
// the problem, and it needs to be pulled and processed by hand.
func (i *Issue) ReportError(message string) {
	i.Error = message
	i.WorkflowStep = schema.WSUnfixableMetadataError
	i.Unclaim()
}

// Save creates or updates the Issue in the issues table
func (i *Issue) Save() error {
	var op = DB.Operation()
	op.Dbg = Debug
	return i.SaveOp(op)
}

// SaveOp creates or updates the Issue in the issues table with a custom operation
func (i *Issue) SaveOp(op *magicsql.Operation) error {
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
	return op.Err()
}

// serialize prepares struct data to work with the database fields better
func (i *Issue) serialize() {
	i.PageLabelsCSV = strings.Join(i.PageLabels, ",")
	i.WorkflowStepString = string(i.WorkflowStep)
}

// deserialize performs operations necessary to get the database data into a more
// useful Go structure
func (i *Issue) deserialize() {
	i.PageLabels = strings.Split(i.PageLabelsCSV, ",")
	i.WorkflowStep = schema.WorkflowStep(i.WorkflowStepString)
	if i.HumanName == "" {
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

// FindCompletedIssuesReadyForRemoval returns all issues which are be complete
// and no longer needed in our workflow: tied to a closed (live_done) batch and
// ignored by NCA, but still contain a location
func FindCompletedIssuesReadyForRemoval() ([]*Issue, error) {
	var op = DB.Operation()
	op.Dbg = Debug

	var list []*Issue
	var cond = "batch_id IN (SELECT id FROM batches WHERE status = ?) AND ignored = 1 AND location <> ''"
	op.Select("issues", &Issue{}).Where(cond, BatchStatusLiveDone).AllObjects(&list)
	deserializeIssues(list)
	return list, op.Err()
}
