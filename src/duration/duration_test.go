package duration

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	var d, err = Parse("1 month 3 years 2 weeks 4 days")
	if err != nil {
		t.Errorf("Got error parsing simple string: %s", err)
	}

	if d.Years != 3 {
		t.Errorf("Expected 3 years, got %d", d.Years)
	}
	if d.Months != 1 {
		t.Errorf("Expected 1 month, got %d", d.Months)
	}
	if d.Weeks != 2 {
		t.Errorf("Expected 2 weeks, got %d", d.Weeks)
	}
	if d.Days != 4 {
		t.Errorf("Expected 4 days, got %d", d.Days)
	}
}

func TestParseWeird(t *testing.T) {
	var d, err = Parse("1M 3yeAr2d")
	if err != nil {
		t.Errorf("Got error parsing short string: %s", err)
	}

	if d.Years != 3 {
		t.Errorf("Expected 3 years, got %d", d.Years)
	}
	if d.Months != 1 {
		t.Errorf("Expected 1 month, got %d", d.Months)
	}
	if d.Weeks != 0 {
		t.Errorf("Expected 0 weeks, got %d", d.Weeks)
	}
	if d.Days != 2 {
		t.Errorf("Expected 2 days, got %d", d.Days)
	}
}

func TestFromDays(t *testing.T) {
	var tests = map[string]struct {
		days     int
		expected string
	}{
		"Zero":        {days: 0, expected: "0 days"},
		"Simple":      {days: 5, expected: "5 days"},
		"Weeks":       {days: 12, expected: "1 week 5 days"},
		"WeeksNoDays": {days: 21, expected: "3 weeks"},
		"Months":      {days: 75, expected: "2 months 2 weeks 1 day"},
		"ManyMonths":  {days: 361, expected: "11 months 3 weeks 6 days"},
		"Years":       {days: 365*2 + 15, expected: "2 years 2 weeks 1 day"},
		"ManyYears":   {days: 365 * 10, expected: "9 years 11 months 4 weeks 1 day"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var dur = FromDays(tc.days)
			var got = dur.String()
			if got != tc.expected {
				t.Errorf("%d days should have returned %q, but we got %q", tc.days, tc.expected, got)
			}
		})
	}
}

func TestString(t *testing.T) {
	var d, err = Parse("1 month 3 years 2 weeks 4 days")
	if err != nil {
		t.Errorf("Got parsing error: %s", err)
	}

	var norm = "3 years 1 month 2 weeks 4 days"
	if d.String() != norm {
		t.Errorf("Expected normalized string to be %q, but got %q", norm, d.String())
	}
}

func TestParseInvalidUnit(t *testing.T) {
	var _, err = Parse("1 month 3 years 2 weeks 4 dayos")
	if err == nil {
		t.Errorf("Expected parsing error, but got nil")
	}
}

func TestParseTooManyUnits(t *testing.T) {
	var _, err = Parse("1 month 3 years 2 months")
	if err == nil {
		t.Errorf("Expected parsing error, but got nil")
	}
	var expected = "months specified more than once"
	var actual = err.Error()
	if expected != actual {
		t.Errorf("Expected error %q, but got %q", expected, actual)
	}
}

func TestEmptyString(t *testing.T) {
	var d, _ = Parse("0y")
	var norm = "0 days"
	if d.String() != norm {
		t.Errorf("Expected normalized string to be %q, but got %q", norm, d.String())
	}

	d, _ = Parse("")
	if d.String() != norm {
		t.Errorf("Expected normalized string to be %q, but got %q", norm, d.String())
	}
}

func TestZero(t *testing.T) {
	var d Duration

	if !d.Zero() {
		t.Errorf("Empty Duration should have Zero() == true")
	}

	d.Days = 1
	if d.Zero() {
		t.Errorf("Duration of one day should have Zero() == false")
	}

	d.Days = 0
	d.Weeks = 1
	if d.Zero() {
		t.Errorf("Duration of one week should have Zero() == false")
	}

	d.Weeks = 0
	d.Months = 1
	if d.Zero() {
		t.Errorf("Duration of one month should have Zero() == false")
	}

	d.Months = 0
	d.Years = 1
	if d.Zero() {
		t.Errorf("Duration of one year should have Zero() == false")
	}
}

func TestInvalidFormats(t *testing.T) {
	var _, err = Parse("one year")
	if err == nil {
		t.Errorf("Expected parsing error, but got nil")
	}

	_, err = Parse("1")
	if err == nil {
		t.Errorf("Expected parsing error, but got nil")
	}
}

func TestRFC3339(t *testing.T) {
	var tests = map[string]struct {
		d        Duration
		expected string
	}{
		"Zero":         {Duration{}, "P0D"},
		"YearsWeeks":   {Duration{Years: 1, Weeks: 5}, "P1Y35D"},
		"Months":       {Duration{Months: 1}, "P1M"},
		"Complex":      {Duration{Years: 4, Months: 5, Days: 8, Weeks: 3}, "P4Y5M29D"},
		"WeeksOnlyISO": {Duration{Weeks: 3}, "P3W"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var got = tc.d.RFC3339()
			if got != tc.expected {
				t.Errorf("Expected %#v to be normalized to %q, but got %q", tc.d, tc.expected, got)
			}
		})
	}
}

func TestAddDate(t *testing.T) {
	tests := map[string]struct {
		start    time.Time
		years    int
		months   int
		days     int
		expected time.Time
	}{
		"Jan31 plus 1 month caps at Feb 28th": {
			start:    time.Date(2023, time.January, 31, 12, 30, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2023, time.February, 28, 12, 30, 0, 0, time.UTC),
		},
		"Jan31 plus 1 month on a leap year caps at Feb 29th": {
			start:    time.Date(2024, time.January, 31, 12, 30, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2024, time.February, 29, 12, 30, 0, 0, time.UTC),
		},
		"Mar31 minus 1 month caps at Feb 28th": {
			start:    time.Date(2023, time.March, 31, 12, 30, 0, 0, time.UTC),
			months:   -1,
			expected: time.Date(2023, time.February, 28, 12, 30, 0, 0, time.UTC),
		},
		"Mar31 minus 1 month leap year caps at Feb 29th": {
			start:    time.Date(2024, time.March, 31, 12, 30, 0, 0, time.UTC),
			months:   -1,
			expected: time.Date(2024, time.February, 29, 12, 30, 0, 0, time.UTC),
		},
		"May31 plus 1 month": {
			start:    time.Date(2023, time.May, 31, 12, 30, 0, 0, time.UTC),
			months:   1,
			expected: time.Date(2023, time.June, 30, 12, 30, 0, 0, time.UTC),
		},
		"Feb29 minus 1 year": {
			start:    time.Date(2024, time.February, 29, 12, 30, 0, 0, time.UTC),
			years:    -1,
			expected: time.Date(2023, time.February, 28, 12, 30, 0, 0, time.UTC),
		},
		"Feb28 plus 1 year leap year": {
			start:    time.Date(2023, time.February, 28, 12, 30, 0, 0, time.UTC),
			years:    1,
			expected: time.Date(2024, time.February, 28, 12, 30, 0, 0, time.UTC),
		},
		"Jan31 plus 1 month plus 5 days": {
			start:    time.Date(2023, time.January, 31, 12, 30, 0, 0, time.UTC),
			months:   1,
			days:     5,
			expected: time.Date(2023, time.March, 5, 12, 30, 0, 0, time.UTC),
		},
		"Zero duration doesn't change anything": {
			start:    time.Date(2023, time.July, 15, 12, 30, 0, 0, time.UTC),
			expected: time.Date(2023, time.July, 15, 12, 30, 0, 0, time.UTC),
		},
		"Simple day add": {
			start:    time.Date(2023, time.July, 15, 12, 30, 0, 0, time.UTC),
			days:     5,
			expected: time.Date(2023, time.July, 20, 12, 30, 0, 0, time.UTC),
		},
		"Add 365 days, get exactly one year forward": {
			start:    time.Date(2022, time.July, 15, 12, 30, 0, 0, time.UTC),
			days:     365,
			expected: time.Date(2023, time.July, 15, 12, 30, 0, 0, time.UTC),
		},
		"Add 365 days to pass a leap day, get +1 year, but -1 day": {
			start:    time.Date(2023, time.July, 15, 12, 30, 0, 0, time.UTC),
			days:     365,
			expected: time.Date(2024, time.July, 14, 12, 30, 0, 0, time.UTC),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := addDate(tc.start, tc.years, tc.months, tc.days)
			if !got.Equal(tc.expected) {
				t.Errorf("addDate(%s, %dy, %dm, %dd): expected %s, but got %s",
					tc.start.Format(time.RFC3339), tc.years, tc.months, tc.days,
					tc.expected.Format(time.RFC3339), got.Format(time.RFC3339))
			}
		})
	}
}

func TestAddToTime(t *testing.T) {
	var jan15 = time.Date(2023, time.January, 15, 0, 0, 0, 0, time.UTC)
	var jan31 = time.Date(2023, time.January, 31, 0, 0, 0, 0, time.UTC)
	var feb29 = time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC) // Leap year

	tests := map[string]struct {
		startTime time.Time
		d         Duration
		expected  time.Time
	}{
		"Zero":              {startTime: jan15, d: Duration{}, expected: jan15},
		"DaysOnly":          {startTime: jan15, d: Duration{Days: 5}, expected: time.Date(2023, time.January, 20, 0, 0, 0, 0, time.UTC)},
		"WeeksOnly":         {startTime: jan15, d: Duration{Weeks: 2}, expected: time.Date(2023, time.January, 29, 0, 0, 0, 0, time.UTC)},
		"MonthsOnly Jan31":  {startTime: jan31, d: Duration{Months: 1}, expected: time.Date(2023, time.February, 28, 0, 0, 0, 0, time.UTC)},
		"MonthsOnly Jan15":  {startTime: jan15, d: Duration{Months: 1}, expected: time.Date(2023, time.February, 15, 0, 0, 0, 0, time.UTC)},
		"YearsOnly Jan31":   {startTime: jan31, d: Duration{Years: 2}, expected: time.Date(2025, time.January, 31, 0, 0, 0, 0, time.UTC)},
		"LeapYearAdd Feb29": {startTime: feb29, d: Duration{Years: 1}, expected: time.Date(2025, time.February, 28, 0, 0, 0, 0, time.UTC)},
		"Complex Jan31":     {startTime: jan31, d: Duration{Years: 1, Months: 1, Weeks: 1, Days: 1}, expected: time.Date(2024, time.March, 8, 0, 0, 0, 0, time.UTC)},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var got = tc.d.AddToTime(tc.startTime)
			if !got.Equal(tc.expected) {
				t.Errorf("Duration %#v from %s: expected %s, but got %s", tc.d, tc.startTime.Format(time.RFC3339), tc.expected.Format(time.RFC3339), got.Format(time.RFC3339))
			}
		})
	}
}

func TestSubtractFromTime(t *testing.T) {
	var mar31 = time.Date(2023, time.March, 31, 0, 0, 0, 0, time.UTC)
	var mar15 = time.Date(2023, time.March, 15, 0, 0, 0, 0, time.UTC)
	var feb29 = time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		startTime time.Time
		d         Duration
		expected  time.Time
	}{
		"Zero":              {startTime: mar15, d: Duration{}, expected: mar15},
		"DaysOnly":          {startTime: mar15, d: Duration{Days: 5}, expected: time.Date(2023, time.March, 10, 0, 0, 0, 0, time.UTC)},
		"WeeksOnly":         {startTime: mar15, d: Duration{Weeks: 2}, expected: time.Date(2023, time.March, 1, 0, 0, 0, 0, time.UTC)},
		"MonthsOnly Mar31":  {startTime: mar31, d: Duration{Months: 1}, expected: time.Date(2023, time.February, 28, 0, 0, 0, 0, time.UTC)},
		"MonthsOnly Mar15":  {startTime: mar15, d: Duration{Months: 1}, expected: time.Date(2023, time.February, 15, 0, 0, 0, 0, time.UTC)},
		"YearsOnly Mar31":   {startTime: mar31, d: Duration{Years: 2}, expected: time.Date(2021, time.March, 31, 0, 0, 0, 0, time.UTC)},
		"LeapYearSub Feb29": {startTime: feb29, d: Duration{Years: 1}, expected: time.Date(2023, time.February, 28, 0, 0, 0, 0, time.UTC)},
		"Complex Mar31":     {startTime: mar31, d: Duration{Years: 1, Months: 1, Weeks: 1, Days: 1}, expected: time.Date(2022, time.February, 20, 0, 0, 0, 0, time.UTC)},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var got = tc.d.SubtractFromTime(tc.startTime)
			if !got.Equal(tc.expected) {
				t.Errorf("Duration %#v ago from %s: expected %s, but got %s", tc.d, tc.startTime.Format(time.RFC3339), tc.expected.Format(time.RFC3339), got.Format(time.RFC3339))
			}
		})
	}
}
