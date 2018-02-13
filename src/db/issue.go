package db

import (
	"fmt"
	"path/filepath"
	"schema"
	"strconv"
	"strings"
	"time"
)

// Workflow steps in-process issues may have - these MUST match the allowed
// values in the database
var allowedWorkflowSteps = []schema.WorkflowStep{
	schema.WSAwaitingProcessing,
	schema.WSAwaitingPageReview,
	schema.WSReadyForMetadataEntry,
	schema.WSAwaitingMetadataReview,
	schema.WSReadyForMETSXML,
	schema.WSReadyForBatching,
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

	// This field is a bit confusing, but it is the NDNP field for the issue
	// "number", which is actually a string since it can contain things like
	// "ISSUE XIX"
	Issue         string
	Edition       int
	EditionLabel  string
	PageLabelsCSV string
	PageLabels    []string `sql:"-"`

	/* Workflow information to keep track of the issue and what it needs */

	Error                  string              // If set, a metadata curator reported a problem
	Location               string              // Where is this issue on disk?
	HumanName              string              // What is the issue's "human" name (for consistent folder naming)?
	IsFromScanner          bool                // Is the issue scanned in-house?  (Born-digital == false)
	HasDerivatives         bool                // Does the issue have derivatives done?
	WorkflowStepString     string              `sql:"workflow_step"` // If set, tells us what "human workflow" step we're on
	WorkflowStep           schema.WorkflowStep `sql:"-"`
	WorkflowOwnerID        int                 // Whose "desk" is this currently on?
	WorkflowOwnerExpiresAt time.Time           // When does the workflow owner lose ownership?
	MetadataEntryUserID    int                 // Who entered metadata?
	ReviewedByUserID       int                 // Who reviewed metadata?
	MetadataApprovedAt     time.Time           // When was metadata approved / how long has this been waiting to batch?
	RejectionNotes         string              // If rejected (during metadata review), this tells us why
	RejectedByUserID       int                 // Who did the rejection?
}

// NewIssue creates an issue ready for saving to the issues table
func NewIssue(moc, lccn, dt string, ed int) *Issue {
	return &Issue{MARCOrgCode: moc, LCCN: lccn, Date: dt, Edition: ed, WorkflowStep: schema.WSAwaitingProcessing}
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

	var op = DB.Operation()
	op.Dbg = Debug

	var list []*Issue
	op.Select("issues", &Issue{}).Where("lccn = ? AND date = ? AND edition = ?", lccn, date, ed).AllObjects(&list)
	deserializeIssues(list)
	return list, op.Err()
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

// NewIssueFromScanDir attempts to take a path and create a DB Issue from it,
// with the assumption that it's from an in-house scan.  This means the
// directory will contain the issue date/edition as the last component, an LCCN
// directory at second-to-last, and a MARC org code just before that.  It is
// assumed that the MOC and LCCN have already been validated.  The issue
// directory name itself will be parsed for validity and an error is returned
// if any part of it is invalid.  If the database already has an issue with the
// same issue key but different data, an error will be returned.
func NewIssueFromScanDir(path string) (*Issue, error) {
	var parts = strings.Split(path, string(filepath.Separator))
	var last = len(parts) - 1
	var moc, lccn, dted = parts[last-2], parts[last-1], parts[last]

	// Make sure the date (and edition, if present) are valid
	var edition = 1

	var dt = dted
	if len(dted) == 13 && dted[10] == '_' {
		var edstr string
		dt, edstr = dted[:10], dted[11:]
		edition, _ = strconv.Atoi(edstr)
		if edition == 0 {
			return nil, fmt.Errorf("invalid edition value")
		}
	}

	var t, err = time.Parse("2006-01-02", dt)
	if err != nil {
		return nil, fmt.Errorf("invalid date directory %q: %s", dted, err)
	}
	var tstr = t.Format("2006-01-02")
	if tstr != dt {
		return nil, fmt.Errorf("invalid date directory %q: time portion parses to %s", dted, tstr)
	}

	var i = NewIssue(moc, lccn, dt, edition)
	i.Location = path
	i.IsFromScanner = true

	// Check for a dupe with the side-effect of extra validation
	var si *schema.Issue
	si, err = i.SchemaIssue()
	if err != nil {
		return nil, err
	}
	var di *Issue
	di, err = FindIssueByKey(si.Key())
	if err != nil {
		return nil, fmt.Errorf("unable to check for dupe of %q: %s", si.Key(), err)
	}

	if di != nil {
		if i.MARCOrgCode != di.MARCOrgCode || i.Location != di.Location || i.IsFromScanner != di.IsFromScanner {
			return nil, fmt.Errorf("existing issue in database (id %d) doesn't match new issue", di.ID)
		}
		return di, nil
	}

	err = i.Save()
	return i, err
}

// FindInProcessIssues returns all issues which have been entered in the
// workflow system, but haven't yet gone through all the way to the point of
// being batched and approved for production
//
// TODO: At the moment, this returns all issues in the database.  This will be
// a problem until the batch maker is migrated *and* we have some way to tie an
// issue to a DB batch **AND** we have a way to flag a batch as being approved
// for prod....  Obviously this MUST be addressed shortly after pushing live;
// it's just a stopgap due to continuing filesystem problems on the legacy
// setup.
func FindInProcessIssues() ([]*Issue, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var list []*Issue
	op.Select("issues", &Issue{}).AllObjects(&list)
	deserializeIssues(list)
	return list, op.Err()
}

// FindIssuesOnDesk returns all issues "owned" by a given user id
func FindIssuesOnDesk(userID int) ([]*Issue, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var list []*Issue
	var sel = op.Select("issues", &Issue{})
	sel = sel.Where(`
		workflow_owner_id = ? AND
		workflow_owner_expires_at IS NOT NULL AND
		workflow_owner_expires_at > ?`, userID, time.Now())
	sel.AllObjects(&list)
	deserializeIssues(list)
	return list, op.Err()
}

// FindIssuesInPageReview looks for all issues currently awaiting page review
// and returns them
func FindIssuesInPageReview() ([]*Issue, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var list []*Issue
	op.Select("issues", &Issue{}).Where("workflow_step = ?", string(schema.WSAwaitingPageReview)).AllObjects(&list)
	deserializeIssues(list)
	return list, op.Err()
}

// FindAvailableIssuesByWorkflowStep looks for all "available" issues with the
// requested workflow step and returns them.  We define "available" as:
//
// - No owner (or owner expired)
// - Have not been reported as having errors
func FindAvailableIssuesByWorkflowStep(ws schema.WorkflowStep) ([]*Issue, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var list []*Issue
	op.Select("issues", &Issue{}).Where(
		"workflow_step = ? AND (workflow_owner_id = 0 OR workflow_owner_expires_at < ?) AND error = ''",
		string(ws), time.Now().Format("2006-01-02 15:04:05")).AllObjects(&list)
	deserializeIssues(list)
	return list, op.Err()
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
func (i *Issue) RejectMetadata(reviewerID int, notes string) {
	i.Claim(i.MetadataEntryUserID)
	i.WorkflowStep = schema.WSReadyForMetadataEntry
	i.RejectionNotes = notes
	i.RejectedByUserID = reviewerID
}

// Save creates or updates the Issue in the issues table
func (i *Issue) Save() error {
	var valid bool
	for _, validWS := range allowedWorkflowSteps {
		if string(i.WorkflowStep) == string(validWS) {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("issue doesn't have a valid workflow step")
	}

	i.serialize()
	var op = DB.Operation()
	op.Dbg = Debug
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
}

// deserializeIssues runs deserialize() against all issues in the list
func deserializeIssues(list []*Issue) {
	for _, i := range list {
		i.deserialize()
	}
}

// SchemaIssue returns an extremely over-simplified representation of this
// issue as a schema.Issue instance for ensuring consistent representation of
// things like issue keys.  NOTE: this will hit the database if titles haven't
// already been loaded!
func (i *Issue) SchemaIssue() (*schema.Issue, error) {
	var dt, err = time.Parse("2006-01-02", i.Date)
	if err != nil {
		return nil, fmt.Errorf("invalid time format (%s) in database issue", i.Date)
	}

	LoadTitles()
	var t = LookupTitle(i.LCCN).SchemaTitle()
	if t == nil {
		return nil, fmt.Errorf("missing title for issue ID %d", i.ID)
	}
	var si = &schema.Issue{
		Date:         dt,
		Edition:      i.Edition,
		Title:        t,
		Location:     i.Location,
		MARCOrgCode:  i.MARCOrgCode,
		WorkflowStep: schema.WorkflowStep(i.WorkflowStep),
	}
	return si, nil
}
