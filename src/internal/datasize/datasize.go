// Package datasize implements very simple logic for NCA's needs.
//
// While a lot of the overall API is inspired from other projects, we chose
// *not* to use those.  This is a pretty simple package and easy enough to
// maintain that we prefer it living here where we can easily debug it, hack
// it, etc.  Also, the most promising package has a lot of little oddities that
// make it a bit concerning to add as a long-term dependency:
//   - Has some confusing error messaging that tries to pretend it comes from
//     strconv functions despite the parser not using strconv.
//   - Imports strconv to fake the above-mentioned errors, yet implements its
//     own custom string-to-int parser (like an extremely naive strconv.Atoi),
//     and uses Sprintf for int-to-string conversions
//   - Implements its own constant for max int
//   - Makes it awkward to just get a quick number from a string
//   - Incredibly unnecessary / sloppy uses of "goto"
package datasize

// Datasize is a simple int64 representation of bytes
type Datasize int64

// New converts a human-friendly disk size indication into a valid Datasize
// (int64) using UnmarshalText.
func New(val string) (Datasize, error) {
	var d Datasize
	var err = d.UnmarshalText([]byte(val))
	return d, err
}

// String returns a human-friendly value for a Datasize using MarshalText
func (d *Datasize) String() string {
	var s, _ = d.MarshalText()
	return string(s)
}
