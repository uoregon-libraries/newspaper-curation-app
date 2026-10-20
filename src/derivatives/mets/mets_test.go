package mets

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// metsPages pulls just the page-level MODS data out of a METS document
type metsPages struct {
	DmdSecs []struct {
		ID      string `xml:"ID,attr"`
		Details []struct {
			Type   string `xml:"type,attr"`
			Number string `xml:"number"`
		} `xml:"mdWrap>xmlData>mods>part>detail"`
	} `xml:"dmdSec"`
}

func getwd(t *testing.T) string {
	var wd, err = os.Getwd()
	if err != nil {
		t.Fatalf("Unable to get working directory: %s", err)
	}

	return wd
}

func TestTransformPageLabels(t *testing.T) {
	var wd = getwd(t)
	var tmplPath = filepath.Join(wd, "..", "..", "..", "templates", "xml", "mets.go.html")
	var outFile = filepath.Join(t.TempDir(), "mets.xml")

	// Labels deliberately differ from page sequence numbers: an unlabeled cover
	// page, two numbered pages, and a section-lettered page
	var title = &models.Title{LCCN: "sn12345678", Name: "The Test Times", ValidLCCN: true}
	var issue = &models.Issue{
		LCCN:       title.LCCN,
		Date:       "2020-01-02",
		Edition:    1,
		Title:      title,
		Location:   filepath.Join(wd, "testdata", "issue"),
		PageLabels: []string{"0", "1", "2", "A1"},
	}

	var err = New(tmplPath, outFile, issue, title, time.Now()).Transform()
	if err != nil {
		t.Fatalf("Unable to transform issue: %s", err)
	}

	var data []byte
	data, err = os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("Unable to read METS output: %s", err)
	}

	var doc metsPages
	err = xml.Unmarshal(data, &doc)
	if err != nil {
		t.Fatalf("Unable to parse METS output: %s", err)
	}

	// Gather each page's label, using an empty string when no page number
	// detail is present
	var got []string
	for _, sec := range doc.DmdSecs {
		if !strings.HasPrefix(sec.ID, "pageModsBib") {
			continue
		}

		var label string
		for _, detail := range sec.Details {
			if detail.Type == "page number" {
				label = detail.Number
			}
		}
		got = append(got, label)
	}

	var expected = []string{"", "1", "2", "A1"}
	var diff = cmp.Diff(expected, got)
	if diff != "" {
		t.Fatalf("Page labels don't match (-expected, +got):\n%s", diff)
	}
}
