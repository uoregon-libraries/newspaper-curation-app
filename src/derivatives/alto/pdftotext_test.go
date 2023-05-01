package alto

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDocClean(t *testing.T) {
	var wd, err = os.Getwd()
	if err != nil {
		t.Fatalf("Unable to get working directory: %s", err)
	}

	var data []byte
	data, err = os.ReadFile(filepath.Join(wd, "testdata", "test.xml"))
	if err != nil {
		t.Fatalf("Unable to read test file: %s", err)
	}

	var source Doc
	err = xml.Unmarshal(data, &source)
	if err != nil {
		t.Fatalf("Unable to parse test file: %s", err)
	}

	var cleaned = source.Clean()
	var cleanedXML []byte
	cleanedXML, err = xml.MarshalIndent(cleaned, "", "  ")
	if err != nil {
		t.Fatalf("Unable to re-marshal XML: %s", err)
	}

	var expectedXML []byte
	expectedXML, err = os.ReadFile(filepath.Join(wd, "testdata", "expected.xml"))
	if err != nil {
		t.Fatalf("Unable to read test file: %s", err)
	}

	// vim puts a newline at the end of a file, so we'll inject one into our
	// cleaned XML for the diff
	cleanedXML = append(cleanedXML, '\n')

	var diff = cmp.Diff(string(cleanedXML), string(expectedXML))
	if diff != "" {
		t.Fatalf(diff)
	}
}
