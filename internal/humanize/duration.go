// Package humanize is made for giving human-friendly output for otherwise
// not-so-friendly values - currently just durations
package humanize

import (
	"math"
	"strconv"
	"time"
)

// Duration returns a human-friendly string describing (roughly) the duration
// d. This is a very rough string just to give people an idea how long
// something has been waiting, and as such isn't meant for general use.
func Duration(d time.Duration) string {
	var hours = d.Hours()
	var days = math.Floor((hours + 4) / 24.0)
	var weeks = math.Floor((days + 0.5) / 7)
	var months = math.Floor(days / 30)
	if weeks > 54 {
		return "over a year"
	}
	if weeks > 50 {
		return "about a year"
	}
	if months > 3 {
		return "about " + strconv.Itoa(int(months)) + " months"
	}
	if weeks == 4 {
		return "about a month"
	}
	if weeks > 3 {
		return "about " + strconv.Itoa(int(weeks)) + " weeks"
	}
	if days == 7 {
		return "about a week"
	}
	if days > 3 {
		return "about " + strconv.Itoa(int(days)) + " days"
	}
	if hours >= 23 {
		return "about a day"
	}
	if hours > 1 {
		return "about " + strconv.Itoa(int(hours)) + " hours"
	}
	var minutes = d.Minutes()
	if minutes < 4 {
		return "a few minutes"
	}
	return "about " + strconv.Itoa(int(d.Minutes())) + " minutes"
}
