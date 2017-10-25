package db

import (
	"fmt"
	"path/filepath"
	"schema"
	"strconv"
	"strings"
	"time"
)

// workflowStep semi-restricts values allowed in the Issue.WorkflowStep field
type workflowStep string

// Human workflow steps - these match the allowed values in the database
const (
	WSAwaitingPageReview     workflowStep = "AwaitingPageReview"
	WSReadyForMetadataEntry               = "ReadyForMetadataEntry"
	WSAwaitingMetadataReview              = "AwaitingMetadataReview"
)

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

	Location               string       // Where is this issue on disk?
	IsFromScanner          bool         // Is the issue scanned in-house?  (Born-digital == false)
	HasDerivatives         bool         // Does the issue have derivatives done?
	WorkflowStepString     string       `sql:"workflow_step"` // If set, tells us what "human workflow" step we're on
	WorkflowStep           workflowStep `sql:"-"`
	WorkflowOwnerID        int          // Whose "desk" is this currently on?
	WorkflowOwnerExpiresAt time.Time    // When does the workflow owner lose ownership?
	MetadataEntryUserID    int          // Who entered metadata?
	ReviewedByUserID       int          // Who reviewed metadata?
}

// NewIssue creates an issue ready for saving to the issues table
func NewIssue(moc, lccn, dt string, ed int) *Issue {
	return &Issue{MARCOrgCode: moc, LCCN: lccn, Date: dt, Edition: ed}
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

// FindIssueByKey looks for an issue in the database that has the given issue key
func FindIssueByKey(key string) (*Issue, error) {
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
	var i = &Issue{}
	var ok = op.Select("issues", &Issue{}).Where("lccn = ? AND date = ? AND edition = ?", lccn, date, ed).First(i)
	if !ok {
		return nil, op.Err()
	}
	i.deserialize()
	return i, op.Err()
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

// FindIssuesInPageReview looks for all issues currently awaiting page review
// and returns them
func FindIssuesInPageReview() ([]*Issue, error) {
	var op = DB.Operation()
	op.Dbg = Debug
	var list []*Issue
	op.Select("issues", &Issue{}).Where("workflow_step = ?", string(WSAwaitingPageReview)).AllObjects(&list)
	return list, op.Err()
}

// Save creates or updates the Issue in the issues table
func (i *Issue) Save() error {
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
	i.WorkflowStep = workflowStep(i.WorkflowStepString)
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
		Date:    dt,
		Edition: i.Edition,
		Title:   t,
	}
	return si, nil
}
