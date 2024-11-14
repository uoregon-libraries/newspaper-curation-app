package marc

import (
	"os"
	"path/filepath"
	"testing"
)

func getwd(t *testing.T) string {
	var wd, err = os.Getwd()
	if err != nil {
		t.Fatalf("Unable to get working directory: %s", err)
	}

	return wd
}

func getFile(t *testing.T, name string) *os.File {
	var wd = getwd(t)
	var f, err = os.Open(filepath.Join(wd, "testdata", name))
	if err != nil {
		t.Fatalf("Unable to read test file %q: %s", name, err)
		return nil
	}

	return f
}

func compare(t *testing.T, field, expected, got string) {
	if expected != got {
		t.Errorf("%s should have been %s, got %s", field, expected, got)
	}
}

func TestParseXML(t *testing.T) {
	var tests = map[string]struct {
		file     string
		lccn     string
		title    string
		location string
		language string
	}{
		"collection-wrapped MARC file": {
			file:     "2002260445-UnitedAmerican.mrk",
			lccn:     "2002260445",
			title:    "The united American : a magazine of good citizenchip.",
			location: "Portland, Or.",
			language: "eng",
		},

		"ONI-provided MARC record": {
			file:     "oni-2024240297-NorthDouglasHerald.xml",
			lccn:     "2024240297",
			title:    "North Douglas herald.",
			location: "Drain Or",
			language: "eng",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var f = getFile(t, tc.file)
			var m, err = ParseXML(f)
			if err != nil {
				t.Fatalf("Unable to parse MARC from %q: %s", tc.file, err)
				return
			}

			compare(t, "LCCN", tc.lccn, m.LCCN())
			compare(t, "Title", tc.title, m.Title())
			compare(t, "Location", tc.location, m.Location())
			compare(t, "Language", tc.language, m.Language())
		})
	}
}
