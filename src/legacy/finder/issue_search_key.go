package main

import (
	"fmt"
	"regexp"
	"schema"
	"strconv"
	"time"
)

// validIssueSearchKey defines the format for a minimal issue-key-like search
// string: strict LCCN, strict year, and optional month, day, and edition
var validIssueSearchKey = regexp.MustCompile(`^(\w{8,10})(/\d+)?`)

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

	if groups[2] == "" {
		return key, nil
	}

	// We have a date, so we strip the slash and start parsing out pieces based
	// on date/edition string's length
	var dtstring = groups[2][1:]
	var dtformat = "20060102"

	var l = len(dtstring)
	if l < 4 || l > 10 || l % 2 != 0 {
		return nil, fmt.Errorf("incorrect number of date/edition digits")
	}

	// The regex and date validation mean we can ignore strconv.Atoi errors below
	if l >= 4 {
		key.year, _ = strconv.Atoi(dtstring[:4])
	}
	if l >= 6 {
		key.month, _ = strconv.Atoi(dtstring[4:6])
	}
	if l >= 8 {
		key.day, _ = strconv.Atoi(dtstring[6:8])
	}
	if l == 10 {
		key.ed, _ = strconv.Atoi(dtstring[8:])
		dtstring = dtstring[:8]
		l = 8
	}

	dtformat = dtformat[:l]

	var dt, err = time.Parse(dtformat, dtstring)
	if err != nil {
		return nil, fmt.Errorf("invalid date")
	}
	if dt.Format(dtformat) != dtstring {
		return nil, fmt.Errorf("date string is non-canonical")
	}

	return key, nil
}

// String returns the textual representation of this search key for use in lookups
func (k issueSearchKey) String() string {
	var keyString = fmt.Sprintf("%s", k.lccn)
	if k.year > 0 {
		keyString += fmt.Sprintf("/%04d", k.year)
	}
	if k.month > 0 {
		keyString += fmt.Sprintf("%02d", k.month)
	}
	if k.day > 0 {
		keyString += fmt.Sprintf("%02d", k.day)
	}
	if k.ed > 0 {
		keyString += fmt.Sprintf("%02d", k.ed)
	}

	return keyString
}

// getLookup returns the appropriate issue map to use when looking up
// issues using this key
func (k *issueSearchKey) getLookup() issueMap {
	if k.year == 0 {
		return issueLookupNoYear
	}
	if k.month == 0 {
		return issueLookupNoMonth
	}
	if k.day == 0 {
		return issueLookupNoDay
	}
	if k.ed == 0 {
		return issueLookupNoEdition
	}
	return issueLookup
}

// issues returns all issues cached using the appropriate lookup and this key
func (k *issueSearchKey) issues() []*schema.Issue {
	var lookup = k.getLookup()
	return lookup[k.String()]
}

// issueKeys returns unique issue keys for this key's lookup.  When reporting,
// we want to first figure out what the search found, then drill into each
// issue key to see what locations that key was seen.
func (k *issueSearchKey) issueKeys() []string {
	var issues = k.issues()
	var keys []string
	var seen = make(map[string]bool)
	for _, i := range issues {
		var ik = i.Key()
		if seen[ik] {
			continue
		}
		seen[ik] = true
		keys = append(keys, ik)
	}

	return keys
}
