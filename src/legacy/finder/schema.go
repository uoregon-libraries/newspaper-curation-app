// schema.go: simple data types for our title and issue finding code to use,
// isolated here so we can more easily reuse this if it makes sense later.

package main

import (
	"fmt"
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

// Title represents whatever common data we need across titles we read from
// filesystem data, database data, and the live site
type Title struct {
	LCCN   string
	Issues []*Issue
}

// AppendIssue creates an issue under this title, sets up its date and edition
// number, and returns it
func (t *Title) AppendIssue(date time.Time, ed int) *Issue {
	var i = &Issue{Date: date, Edition: ed, Title: t}
	t.Issues = append(t.Issues, i)
	return i
}

// Issue is an extremely basic encapsulation of an issue's high-level data
type Issue struct {
	Date    time.Time
	Edition int
	Title   *Title
	Batch   *Batch
}

// Key returns the unique string that represents this issue
func (i *Issue) Key() string {
	return fmt.Sprintf("%s/%s%02d", i.Title.LCCN, i.Date.Format("20050102"), i.Edition)
}
