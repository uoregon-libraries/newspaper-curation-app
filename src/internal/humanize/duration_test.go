package humanize

import (
	"testing"
	"time"
)

func TestDuration(t *testing.T) {
	var tests = map[string]struct {
		input time.Duration
		want  string
	}{
		"Small":  {time.Second, "a few minutes"},
		"Weeks":  {time.Hour * 24 * 25, "about 3 weeks"},
		"Month":  {time.Hour * 24 * 30, "about a month"},
		"Months": {time.Hour * 24 * 90, "about 3 months"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var got = Duration(tc.input)
			if tc.want != got {
				t.Errorf("Expected %s to give us %q, got %q", tc.input, tc.want, got)
			}
		})
	}
}
