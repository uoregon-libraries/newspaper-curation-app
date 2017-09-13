package db

import (
	"fmt"
	"strconv"
	"strings"
)

// Issue contains metadata about an issue for the various workflow tools' use
//
// TODO: Right now this is only used by the Go tools, but eventually the PHP
// and Python scripts need to use this (or be rewritten) so we are managing
// more of the workflow via data, not JSON files.  Hence all the extra data
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
