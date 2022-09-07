package datasize

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Datasize constants - untyped so they can be used without conversion since
// these are pretty generally-useful values
const (
	B  = 1
	KB = B * 1024
	MB = KB * 1024
	GB = MB * 1024
	TB = GB * 1024
	PB = TB * 1024
	EB = PB * 1024
)

// sizes: all valid datasize increments
var sizes = []Datasize{B, KB, MB, GB, TB, PB, EB}

// UnmarshalText does the dirty work of converting a string representation of
// some number of bytes into a usable Datasize.
//
// Currently this only handles one- and two-character suffixes (e.g., "gb",
// "kb", etc.) and is limited to sizes that can be represented by an int64
// (-8EB to 8EB - 1).
func (d *Datasize) UnmarshalText(data []byte) error {
	var val = string(data)

	// Remove all spaces to allow for things like "100 mb"
	var runes = []rune(strings.Replace(val, " ", "", -1))

	// Split into number and non-numeric parts in a reasonably fast but naive
	// way: we find the first unicode letter and split there.  ParseInt64 tells
	// us when the "number" portion is invalid.
	var numeric, size string
	for i, r := range runes {
		if unicode.IsLetter(r) {
			numeric = string(runes[:i])
			size = strings.ToLower(string(runes[i:]))
			break
		}
	}
	if size == "" {
		numeric = val
	}

	var mult int64
	switch size {
	case "", "b":
		mult = B
	case "k", "kb":
		mult = KB
	case "m", "mb":
		mult = MB
	case "g", "gb":
		mult = GB
	case "t", "tb":
		mult = TB
	case "p", "pb":
		mult = PB
	case "e", "eb":
		mult = EB
	default:
		return fmt.Errorf("%q is not a valid size suffix", size)
	}

	var base, err = strconv.ParseInt(numeric, 10, 64)
	if err != nil {
		return fmt.Errorf("%q is an invalid number: %s", numeric, err)
	}

	var total = base * mult
	if total/mult != base {
		return fmt.Errorf("outside int64 range")
	}

	*d = Datasize(int64(total))
	return nil
}

// MarshalText returns a human-friendly value for a Datasize by getting the
// largest size ("KB", "MB", etc.) that leaves a whole number for the numeric
// portion of the string
func (d Datasize) MarshalText() (text []byte, err error) {
	var numeric = int64(d)
	var suffix = "B"

	switch {
	case d%EB == 0:
		numeric = int64(d / EB)
		suffix = "EB"
	case d%PB == 0:
		numeric = int64(d / PB)
		suffix = "PB"
	case d%TB == 0:
		numeric = int64(d / TB)
		suffix = "TB"
	case d%GB == 0:
		numeric = int64(d / GB)
		suffix = "GB"
	case d%MB == 0:
		numeric = int64(d / MB)
		suffix = "MB"
	case d%KB == 0:
		numeric = int64(d / KB)
		suffix = "KB"
	}

	return []byte(strconv.FormatInt(numeric, 10) + " " + suffix), nil
}
