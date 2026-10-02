package marc

import (
	"bytes"
	"io"
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

// getReader returns a reader for the given test file's contents, optionally
// wrapping the data in a <collection> element
func getReader(t *testing.T, name string, collection bool) io.Reader {
	var f = getFile(t, name)
	defer f.Close()

	var data, err = io.ReadAll(f)
	if err != nil {
		t.Fatalf("Unable to read test file %q: %s", name, err)
	}

	if collection {
		data = append(append([]byte("<collection>"), data...), "</collection>"...)
	}

	return bytes.NewReader(data)
}

func compare(t *testing.T, field, expected, got string) {
	if expected != got {
		t.Errorf("%s should have been %s, got %s", field, expected, got)
	}
}

func TestParseXML(t *testing.T) {
	var tests = map[string]struct {
		file       string
		collection bool
		lccn       string
		title      string
		location   string
		language   string
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

		"MARC record with apostrophe": {
			file:     "peoples-press.mrk",
			lccn:     "2026234394",
			title:    "The people's press.",
			location: "Eugene City, Oregon",
			language: "eng",
		},

		// Wrapping in a collection forces the record to be re-marshaled, which
		// encodes apostrophes as "&#39;"
		"collection-wrapped MARC record with apostrophe": {
			file:       "peoples-press.mrk",
			collection: true,
			lccn:       "2026234394",
			title:      "The people's press.",
			location:   "Eugene City, Oregon",
			language:   "eng",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var m, err = ParseXML(getReader(t, tc.file, tc.collection))
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

func TestParseXMLDecodesEntities(t *testing.T) {
	for _, collection := range []bool{false, true} {
		var m, err = ParseXML(getReader(t, "peoples-press.mrk", collection))
		if err != nil {
			t.Fatalf("Unable to parse MARC (collection: %t): %s", collection, err)
		}

		compare(t, "264$3", "<Nov. 17, 1860-> :", m.Get("264", "3"))
	}
}
