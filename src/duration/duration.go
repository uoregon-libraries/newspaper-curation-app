// Package duration deals with very simple durations similarly to durations
// defined by RFC 3339, but with a very simple scope:
//     - A duration is only granular to the day
//     - Values may have any number of spaces in them, which are ignored
//     - Unit names can be short or long, e.g., "w", "week", or "weeks"
package duration

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var reg = regexp.MustCompile(`(\d+)([a-zA-Z]+)`)

// Duration represents a period of zero or more days, weeks, months, and/or years
type Duration struct {
	Days   int
	Weeks  int
	Months int
	Years  int
}

// unit represents a time unit during parsing
type unit int

const (
	invalid unit = iota
	day
	week
	month
	year
)

// unitMap maps the various versions of a string to the proper unit value
var unitMap = map[string]unit{
	"d":    day,
	"day":  day,
	"days": day,

	"w":     week,
	"week":  week,
	"weeks": week,

	"m":      month,
	"month":  month,
	"months": month,

	"y":     year,
	"year":  year,
	"years": year,
}

// Parse attempts to conver the given string into a Duration.  An invalid
// format will result in an error.
func Parse(s string) (Duration, error) {
	var d Duration

	s = strings.ToLower(strings.Replace(s, " ", "", -1))
	if s == "" {
		return d, nil
	}
	if s == "0" {
		return d, nil
	}

	var groups = reg.FindAllStringSubmatch(s, -1)
	for _, group := range groups {
		var numStr, unit = group[1], group[2]
		var num, _ = strconv.Atoi(numStr)

		var u = unitMap[unit]
		switch u {
		case day:
			if d.Days > 0 {
				return d, fmt.Errorf("days specified more than once")
			}
			d.Days = num

		case week:
			if d.Weeks > 0 {
				return d, fmt.Errorf("weeks specified more than once")
			}
			d.Weeks = num

		case month:
			if d.Months > 0 {
				return d, fmt.Errorf("months specified more than once")
			}
			d.Months = num

		case year:
			if d.Years > 0 {
				return d, fmt.Errorf("years specified more than once")
			}
			d.Years = num

		default:
			return d, fmt.Errorf("invalid unit name %q", unit)
		}
	}

	return d, nil
}

func appendDuration(out []string, num int, unit string) []string {
	if num < 1 {
		return out
	}

	if num == 1 {
		return append(out, "1 "+unit)
	}

	return append(out, strconv.Itoa(num)+" "+unit+"s")
}

func (d Duration) String() string {
	var out []string
	out = appendDuration(out, d.Years, "year")
	out = appendDuration(out, d.Months, "month")
	out = appendDuration(out, d.Weeks, "week")
	out = appendDuration(out, d.Days, "day")

	if len(out) == 0 {
		return "0 days"
	}

	return strings.Join(out, " ")
}

// Zero returns true if the duration represents precisely zero
func (d Duration) Zero() bool {
	return d.Years == 0 && d.Months == 0 && d.Weeks == 0 && d.Days == 0
}
