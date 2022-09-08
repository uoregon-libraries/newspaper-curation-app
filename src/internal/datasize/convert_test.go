package datasize

import (
	"encoding/json"
	"math/rand"
	"testing"
)

func TestJSON(t *testing.T) {
}

func TestUnmarshal(t *testing.T) {
	var tests = map[string]struct {
		input     string
		want      int64
		wantError bool
	}{
		"simple bytes":    {"1024", 1024, false},
		"bytes with a b":  {"1024b", 1024, false},
		"kilobytes":       {"27k", 27 * KB, false},
		"kilobytes 2":     {"27kb", 27 * KB, false},
		"megabytes":       {"10m", 10 * MB, false},
		"megabytes 2":     {"10 m", 10 * MB, false},
		"megabytes 3":     {"10mb", 10 * MB, false},
		"gigabytes":       {"39g", 39 * GB, false},
		"gigabytes 2":     {"39 g", 39 * GB, false},
		"gigabytes 3":     {"39gb", 39 * GB, false},
		"terabytes":       {"12tb", 12 * TB, false},
		"petabytes":       {"1p", PB, false},
		"petabytes 2":     {"1 p", PB, false},
		"petabytes 3":     {"1pb", PB, false},
		"exabytes":        {"1eb", EB, false},
		"8EB is too much": {"8eb", 0, true},
		"-8EB is fine":    {"-8eb", -8 * EB, false},
		"bad input":       {"gigabytes: 3", 0, true},
		"bad input 2":     {"3 gb gb", 0, true},
		"sanity eb-a":     {"7eb", 8070450532247928832, false},
		"sanity eb-b":     {"7eb", 7 * PB * 1024, false},
		"sanity pb-a":     {"7pb", 7881299347898368, false},
		"sanity pb-b":     {"7pb", 7 * TB * 1024, false},
		"sanity tb-a":     {"7tb", 7696581394432, false},
		"sanity tb-b":     {"7tb", 7 * GB * 1024, false},
		"sanity gb-a":     {"7gb", 7516192768, false},
		"sanity gb-b":     {"7gb", 7 * MB * 1024, false},
		"sanity mb-a":     {"7mb", 7340032, false},
		"sanity mb-b":     {"7mb", 7 * KB * 1024, false},
		"sanity kb-a":     {"7kb", 7168, false},
		"sanity kb-b":     {"7kb", 7 * B * 1024, false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var d Datasize
			var err = d.UnmarshalText([]byte(tc.input))
			var got = int64(d)

			if tc.wantError && err == nil {
				t.Errorf("%q should have returned an error", tc.input)
			}
			if !tc.wantError && err != nil {
				t.Errorf("%q should not have returned an error, but got %s", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("%q should have returned %d, but got %d", tc.input, tc.want, got)
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	var tests = map[string]struct {
		input int64
		want  string
	}{
		"simple bytes": {10, "10 B"},
		"1024":         {1024, "1 KB"},
		"kilobytes":    {27 * KB, "27 KB"},
		"megabytes":    {10 * MB, "10 MB"},
		"gigabytes":    {39 * GB, "39 GB"},
		"terabytes":    {12 * TB, "12 TB"},
		"55 megs raw":  {57671680, "55 MB"},
		"negative":     {-57671680, "-55 MB"},
		"petabytes":    {PB, "1 PB"},
		"exabytes":     {EB, "1 EB"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var d = Datasize(tc.input)
			var data, err = d.MarshalText()
			if err != nil {
				t.Errorf("Got error marshaling!  This can't happen!!")
			}

			var got = string(data)
			if got != tc.want {
				t.Errorf("Marshaling %d should give us %q, but got %q", tc.input, tc.want, got)
			}
		})
	}
}

// This might seem unnecessary, but using JSON to test a round-trip means not
// only does round-tripping work, but also we implemented the marshal/unmarshal
// functions correctly
func TestJSONRoundTripper(t *testing.T) {
	var vals = make([]Datasize, 1000)
	var mult = KB
	// Simple tests that hit various whole numbers with a suffix other than "B"
	for i := 0; i < 100; i++ {
		vals[i] = Datasize(mult * (rand.Intn(7000) - 3500))
		if mult == PB {
			mult = KB
		} else {
			mult *= 1024
		}
	}

	for i := 100; i < 1000; i++ {
		// Completely random inputs here
		vals[i] = Datasize(rand.Uint64())
	}

	var jsonData, err = json.Marshal(vals)
	if err != nil {
		t.Fatalf("Unable to marshal vals into JSON: %s", err)
	}
	var newVals []Datasize
	err = json.Unmarshal(jsonData, &newVals)
	if err != nil {
		t.Fatalf("Unable to unmarshal vals into JSON: %s", err)
	}

	for i, v := range vals {
		if v != newVals[i] {
			t.Fatalf("Error with value %d: original values was %d, but after round trip, it was %d", i, v, newVals[i])
		}
	}
}
