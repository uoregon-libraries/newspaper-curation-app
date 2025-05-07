// Package duration deals with very simple durations similarly to durations
// defined by ISO 8601, but with a simpler scope:
//   - A duration is only granular to the day
//   - Values may have any number of spaces in them, which are ignored
//   - Unit names can be short or long, e.g., "w", "week", or "weeks"
package duration

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
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
	if len(groups) == 0 {
		return d, fmt.Errorf("invalid time period")
	}

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

const daysPerYear = 365.2425
const daysPerMonth = daysPerYear / 12

// FromDays lets us pass in a number of days, and "reduces" the value to a
// meaningful duration by extracting a rough estimate of years and months. The
// corresponding duration *will not* be 100% accurate! There are not a set
// number of days in a month, or even in a year. The Duration will be close,
// but cannot be used if precision is necessary.
func FromDays(days int) Duration {
	var d Duration

	// If we have several years' worth of days, we sort of fake leap years.
	if days >= 1460 {
		d.Years += int(float64(days) / daysPerYear)
		days -= int(float64(d.Years) * daysPerYear)
	}

	// Otherwise, just a simple 365 division
	if days >= 365 {
		d.Years += days / 365
		days %= 365
	}

	// If we have a lot of days left, we calculate months sort of accurately
	if days >= 183 {
		d.Months += int(float64(days) / daysPerMonth)
		days -= int(float64(d.Months) * daysPerMonth)
	}

	// Otherwise, just 30
	if days >= 30 {
		d.Months += days / 30
		days %= 30
	}

	if days >= 7 {
		d.Weeks += days / 7
		days %= 7
	}

	d.Days = days

	return d
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

// RFC3339 returns an unambiguous, machine-friendly string containing one or
// more groups of number + single-letter unit, uppercased
func (d Duration) RFC3339() string {
	if d.Zero() {
		return "P0D"
	}

	if d.Weeks > 0 && d.Years == 0 && d.Months == 0 && d.Days == 0 {
		return "P" + strconv.Itoa(d.Weeks) + "W"
	}

	var s = "P"
	var inf = []struct {
		num  int
		unit string
	}{
		{d.Years, "Y"},
		{d.Months, "M"},
		{d.Days + d.Weeks*7, "D"},
	}
	for _, i := range inf {
		if i.num > 0 {
			s += strconv.Itoa(i.num) + i.unit
		}
	}

	return s
}

// Zero returns true if the duration represents precisely zero
func (d Duration) Zero() bool {
	return d.Years == 0 && d.Months == 0 && d.Weeks == 0 && d.Days == 0
}

// addDate adds the specified number of years, months, and days to time t.
// Unlike [time.AddDate], we want to handle month-end rollovers more
// intuitively.
//
// For days, this isn't an issue. People expect when you add X days, you get a
// precise number of days forward. Simple.
//
// Adding months or years is less simple. If it's Feb 29th, it's confusing if
// "+1 year" results in March 1st. If it's July 31st, adding 2 months shouldn't
// result in October first. People's brains aren't wired this way!
func addDate(t time.Time, years, months, days int) time.Time {
	// First we calculate the target month / year as explained above: avoid
	// normalizing when year and month are changed
	var currentYear, currentMonth, _ = t.Date()
	var targetMonthInt = int(currentMonth) + months
	var targetYear = currentYear + years
	targetYear += (targetMonthInt - 1) / 12
	targetMonthInt = (targetMonthInt-1)%12 + 1
	if targetMonthInt <= 0 {
		targetMonthInt += 12
		targetYear--
	}
	var targetMonth = time.Month(targetMonthInt)

	// Next we get the last day of the calculated target month by creating a date
	// one month after, and subtracting a day.
	var firstOfFollowingMonth = time.Date(targetYear, targetMonth+1, 1, 0, 0, 0, 0, t.Location())
	var lastDayOfTargetMonth = firstOfFollowingMonth.AddDate(0, 0, -1).Day()

	// Get the original day, capped at the last day of the target month
	var originalDay = t.Day()
	var newDay = originalDay
	if newDay > lastDayOfTargetMonth {
		newDay = lastDayOfTargetMonth
	}

	// Now we build the actual time.Time with t's Day (possibly capped) and the
	// targeted year/month
	var intermediateTime = time.Date(targetYear, targetMonth, newDay,
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())

	// Finally, add the days normally and return it
	return intermediateTime.AddDate(0, 0, days)
}

// AddToTime returns the time.Time corresponding to the given time plus this
// duration. It uses custom logic to handle month-end rollovers more
// intuitively than the standard time.AddDate. Weeks and days are added
// separately after month/year adjustments.
func (d Duration) AddToTime(t time.Time) time.Time {
	return addDate(t, d.Years, d.Months, d.Weeks*7+d.Days)
}

// SubtractFromTime returns the time.Time corresponding to the given time minus
// this duration. It uses custom logic to handle month-end rollovers more
// intuitively than the standard time.AddDate. Weeks and days are subtracted
// separately after month/year adjustments.
func (d Duration) SubtractFromTime(t time.Time) time.Time {
	return addDate(t, -d.Years, -d.Months, -(d.Weeks*7 + d.Days))
}
