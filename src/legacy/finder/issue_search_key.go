package main

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// validIssueSearchKey defines the format for a minimal issue-key-like search
// string: strict LCCN, strict year, and optional month, day, and edition
var validIssueSearchKey = regexp.MustCompile(`^(\w{10})/(\d{4})(\d\d)?(\d\d)?(\d\d)?$`)

// issueSearchKey defines the precise issue (or subset of issues) we want to
// find.  Note that the structure here is very specific to this issue finder,
// so we don't expect (or even want) reuse.
type issueSearchKey struct {
	source string
	lccn   string
	year   int
	month  int
	day    int
	ed     int
}

// parseSearchKey attempts to read the given string, returning an error if the
// string isn't a valid search key, otherwise returning a proper issueSearchKey
func parseSearchKey(ik string) (*issueSearchKey, error) {
	var groups = validIssueSearchKey.FindStringSubmatch(ik)
	if groups == nil {
		return nil, fmt.Errorf("invalid issue key format")
	}
	var key = &issueSearchKey{source: ik, lccn: groups[1]}

	// Validate whatever parts of the date we can
	var dtstring = groups[2] + groups[3] + groups[4]
	var dtformat = "2006"
	if groups[3] != "" {
		dtformat += "01"
	}
	if groups[4] != "" {
		dtformat += "02"
	}
	var dt, err = time.Parse(dtformat, dtstring)
	if err != nil {
		return nil, fmt.Errorf("invalid date")
	}
	if dt.Format(dtformat) != dtstring {
		return nil, fmt.Errorf("date string is non-canonical")
	}

	// The regex and date validation mean we can ignore errors in strconv.Atoi
	key.year, _ = strconv.Atoi(groups[2])
	key.month, _ = strconv.Atoi(groups[3])
	key.day, _ = strconv.Atoi(groups[4])
	key.ed, _ = strconv.Atoi(groups[5])

	return key, nil
}

