package main

import (
	"testing"
)

func TestParseBatchname(t *testing.T) {
	var name = "batch_oru_fluffythedog_ver02"
	var b, err = ParseBatchname(name)
	if err != nil {
		t.Fatalf("Error parsing valid batch name: %s", err)
	}

	if b.Fullname() != name {
		t.Fatalf("b.Fullname() (%#v) doesn't match our input value", err)
	}

	if b.Version != 2 {
		t.Fatalf("Batch %#v: version wasn't 2", b)
	}
}

func TestParseNonconformingToSpecBatchname(t *testing.T) {
	var name = "batch_oru_courage_3_ver01"
	var b, err = ParseBatchname(name)
	if err != nil {
		t.Fatalf("Error parsing valid batch name (yes I know it violates the spec, " +
			"but it's still considered valid for some awful reason): %s", err)
	}

	if b.Fullname() != name {
		t.Fatalf("b.Fullname() (%#v) doesn't match our input value", err)
	}

	if b.MARCOrgCode != "oru" {
		t.Fatalf(`b.MARCOrgCode (%#v) should have been "oru"`, b.MARCOrgCode)
	}
	if b.Keyword != "courage_3" {
		t.Fatalf(`b.Keyword (%#v) should have been "courage_3"`, b.Keyword)
	}

	if b.Version != 1 {
		t.Fatalf("Batch %#v: version wasn't 1", b)
	}
}
