package db

import (
	"fmt"
	"strconv"
	"strings"
)

// WorkflowStep is an enumeration of where an issue can be in the workflow
type WorkflowStep int

const (
	// WSUnknown is the zero-value workflow step, indicating something wasn't
	// initialized properly
	WSUnknown WorkflowStep = iota

	// WSPreppingSFTPIssueForMove is set when a born-digital issue is about to be
	// moved for processing so we can track what happened if it fails to move
	WSPreppingSFTPIssueForMove

	// WSAwaitingPDFProcessing is for born-digital issues that have just been
	// moved from their upload location and need to be split, converted to PDF/a,
	// and moved on for page reordering.
	WSAwaitingPDFProcessing

	// WSAwaitingManualProcessing is used only for born-digital issues as they
	// need to be reordered and sometimes have corrections, such as removing
	// duplicated pages
	WSAwaitingManualProcessing

	// WSReadyToProcess means the issues are 100% in the control of our
	// applications, and can have derivatives created, metadata entered, etc.
	WSReadyToProcess

	// WSReadyToBatch is set up once derivatives have been created, and all the
	// metadata is entered and reviewed
	WSReadyToBatch
)

// Issue contains data about an issue - metadata as well as workflow and
// location.  This allows us to know where an issue is in the process, what
// still needs to be done, who currently "owns" it, etc.  It also lets us track
// on issues that have are waiting for manual processing so we know to check if
// they've been dealt with.
//
// TODO: Right now this is only used by the Go tools, but eventually the PHP
// and Python scripts need to use this (or be rewritten) so we are managing
// more of the workflow via data, not directories.  Hence all the extra data
// which these tools currently don't use.
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

	// Workflow data
	Location         string
	WorkflowStep     WorkflowStep
	NeedsDerivatives bool
	Status           string
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

// NewIssue sets up a structure for storing issue metadata in the database
func NewIssue(location string) *Issue {
	return &Issue{Location: location, NeedsDerivatives: true}
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
}

// deserialize performs operations necessary to get the database data into a more
// useful Go structure
func (i *Issue) deserialize() {
	i.PageLabels = strings.Split(i.PageLabelsCSV, ",")
}
