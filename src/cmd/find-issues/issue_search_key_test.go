package main

import (
	"fmt"
	"strings"
	"testing"
)

func expectError(testName, ik, errStr string, t *testing.T) {
	var _, err = parseSearchKey(ik)
	if err == nil {
		t.Fatalf("[%s] No error returned, expected error with %#v in string", testName, errStr)
	}

	if !strings.Contains(err.Error(), errStr) {
		t.Fatalf("[%s] Error mismatch: expected error with %#v in string, got %#v", testName, errStr, err.Error())
	}
}

func expectKey(testName, ik, lccn string, year, month, day, edition int, t *testing.T) {
	var key, err = parseSearchKey(ik)
	if err != nil {
		t.Fatalf("[%s] Error parsing search key %#v: %s", testName, ik, err)
	}

	var errors []string
	if key.lccn != lccn {
		errors = append(errors, fmt.Sprintf("expected LCCN %#v", lccn))
	}
	if key.year != year {
		errors = append(errors, fmt.Sprintf("expected year %d", year))
	}
	if key.month != month {
		errors = append(errors, fmt.Sprintf("expected month %d", month))
	}
	if key.day != day {
		errors = append(errors, fmt.Sprintf("expected day %d", day))
	}
	if key.ed != edition {
		errors = append(errors, fmt.Sprintf("expected edition %d", edition))
	}

	if len(errors) != 0 {
		t.Fatalf("[%s] Error in key %#v <%#v> (%s)", testName, ik, key, strings.Join(errors, ", "))
	}
}

func TestFullSearchKey(t *testing.T) {
	expectKey("full search key", "sn12345678/2011050201", "sn12345678", 2011, 5, 2, 1, t)
}

func TestPartialSearchKeys(t *testing.T) {
	expectKey("year-only key", "sn12345678/2011", "sn12345678", 2011, 0, 0, 0, t)
	expectKey("year-month-only key", "sn12345678/201105", "sn12345678", 2011, 5, 0, 0, t)
	expectKey("year-month-day-only key", "sn12345678/20110502", "sn12345678", 2011, 5, 2, 0, t)
}

func TestInvalidSearchKey(t *testing.T) {
	expectError("Key with bad month", "sn12345678/2000019901", "invalid date", t)
	expectError("Key with weird date", "sn12345678/2000010001", "date string is non-canonical", t)
}
