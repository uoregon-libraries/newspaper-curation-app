// Package schema houses simple data types for titles, issues, batches, etc.
// Types which live here are generally meant to be very general-case rather
// than trying to hold all possible information for all possible use cases.
//
// Except... a Location field exists on all structures because the workflow
// allows for multiple occurrences of metadata for any of the schema items.
// They could be on the filesystem or the web.  And in the case of errors,
// which we need to be able to detect, there can be dupes that need to be
// reported and figured out.
package schema

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Batch represents high-level batch information
type Batch struct {
	// MARCOrgCode tells us the organization responsible for the images in the batch
	MARCOrgCode string

	// A batch's keyword is normally short, such as "horsetail", but our in-house
	// batches have much longer keywords to ensure uniqueness
	Keyword string

	// Usually 1, but I've seen "_ver02" batches occasionally
	Version int

	// Issues links the issues which are part of this batch
	Issues []*Issue

	// Location is where this batch can be found, either a URL or filesystem path
	Location string
}

// ParseBatchname creates a Batch by splitting up the full name string
func ParseBatchname(fullname string) (*Batch, error) {
	// All batches must have the format "batch_MARCORGCODE_NAME_ver##"
	var parts = strings.Split(fullname, "_")

	// This is really obnoxious, but we can only test for too few parse.  Despite
	// the spec's claim that the batch keyword must not have underscores, some
	// live batches do.  I'm lookin' at you, "courage_3".
	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid batch format")
	}

	if parts[0] != "batch" {
		return nil, fmt.Errorf(`batches must begin with "batch_"`)
	}

	var l = len(parts)
	var b = &Batch{}
	var ver string
	parts, ver = parts[:l-1], parts[l-1]
	b.MARCOrgCode, b.Keyword = parts[1], strings.Join(parts[2:], "_")

	if len(ver) != 5 || ver[:3] != "ver" {
		return nil, fmt.Errorf("invalid version format")
	}

	b.Version, _ = strconv.Atoi(ver[3:])
	if b.Version < 1 {
		return nil, fmt.Errorf("invalid version value")
	}

	return b, nil
}

// Fullname is the full batch name
func (b *Batch) Fullname() string {
	var parts = []string{"batch", b.MARCOrgCode, b.Keyword, fmt.Sprintf("ver%02d", b.Version)}
	return strings.Join(parts, "_")
}

// AddIssue adds the issue to this batch's list, and sets the issue's batch
func (b *Batch) AddIssue(i *Issue) {
	b.Issues = append(b.Issues, i)
	i.Batch = b
}

// Title is a very simple structure to give us something common we can tie to
// anything with the same LCCN
type Title struct {
	LCCN  string
}

// Issue is an extremely basic encapsulation of an issue's high-level data
type Issue struct {
	Title   *Title
	Date    time.Time
	Edition int
	Batch   *Batch

	// Location is where this issue can be found, either a URL or filesystem path
	Location string
}

// Key returns the unique string that represents this issue
func (i *Issue) Key() string {
	return fmt.Sprintf("%s/%s%02d", i.Title.LCCN, i.Date.Format("20060102"), i.Edition)
}

// IssueList groups a bunch of issues together
type IssueList []*Issue

// SortByKey modifies the IssueList in place so they're sorted alphabetically
// by issue key
func (list IssueList) SortByKey() {
	sort.Slice(list, func(i, j int) bool { return list[i].Key() < list[j].Key() })
}
