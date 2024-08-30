package duration

import "testing"

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
	var tests = []struct {
		d        Duration
		expected string
	}{
		{Duration{}, "P0D"},
		{Duration{Years: 1, Weeks: 5}, "P1Y35D"},
		{Duration{Months: 1}, "P1M"},
		{Duration{Years: 4, Months: 5, Days: 8, Weeks: 3}, "P4Y5M29D"},
		{Duration{Weeks: 3}, "P3W"},
	}

	for _, test := range tests {
		if test.d.RFC3339() != test.expected {
			t.Errorf("Expected %#v to be normalized to %q, but got %q", test.d, test.expected, test.d.RFC3339())
		}
	}
}
