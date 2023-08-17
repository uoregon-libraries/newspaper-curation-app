package schema

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// validKey defines the format for a minimal issue-key-like search
// string: LCCN, year, month, day, and edition
var validKey = regexp.MustCompile(`^(\w+)(/\d+)?$`)

// Key defines the precise issue (or subset of issues) we want to
// find.  Note that the structure here is very specific to this issue finder,
// so we don't expect (or even want) reuse.
type Key struct {
	Source string
	LCCN   string
	Year   int
	Month  int
	Day    int
	Ed     int
}

// ParseSearchKey attempts to read the given string, returning an error if the
// string isn't a valid search key, otherwise returning a proper issueSearchKey
func ParseSearchKey(ik string) (*Key, error) {
	var groups = validKey.FindStringSubmatch(ik)
	if groups == nil {
		return nil, fmt.Errorf("invalid issue key format")
	}
	var key = &Key{Source: ik, LCCN: groups[1]}

	if groups[2] == "" {
		return key, nil
	}

	// We have a date, so we strip the slash and start parsing out pieces based
	// on date/edition string's length
	var dtstring = groups[2][1:]
	var dtformat = "20060102"

	var l = len(dtstring)
	if l < 4 || l > 10 || l%2 != 0 {
		return nil, fmt.Errorf("incorrect number of date/edition digits")
	}

	// The regex and date validation mean we can ignore strconv.Atoi errors below
	if l >= 4 {
		key.Year, _ = strconv.Atoi(dtstring[:4])
	}
	if l >= 6 {
		key.Month, _ = strconv.Atoi(dtstring[4:6])
	}
	if l >= 8 {
		key.Day, _ = strconv.Atoi(dtstring[6:8])
	}
	if l == 10 {
		key.Ed, _ = strconv.Atoi(dtstring[8:])
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
func (k Key) String() string {
	var keyString = k.LCCN
	if k.Year > 0 {
		keyString += fmt.Sprintf("/%04d", k.Year)
	}
	if k.Month > 0 {
		keyString += fmt.Sprintf("%02d", k.Month)
	}
	if k.Day > 0 {
		keyString += fmt.Sprintf("%02d", k.Day)
	}
	if k.Ed > 0 {
		keyString += fmt.Sprintf("%02d", k.Ed)
	}

	return keyString
}
