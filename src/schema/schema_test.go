package schema

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

func TestParseBatchname(t *testing.T) {
	var name = "batch_oru_fluffythedog_ver02"
	var b, err = ParseBatchname(name)
	if err != nil {
		t.Fatalf("Error parsing valid batch name: %s", err)
	}

	if b.Fullname() != name {
		t.Fatalf("b.Fullname() (%#v) doesn't match our input value", err)
	}

	if b.Version != 2 {
		t.Fatalf("Batch %#v: version wasn't 2", b)
	}
}

func TestParseNonconformingToSpecBatchname(t *testing.T) {
	var name = "batch_oru_courage_3_ver01"
	var b, err = ParseBatchname(name)
	if err != nil {
		t.Fatalf("Error parsing valid batch name (yes I know it violates the spec, "+
			"but it's still considered valid for some awful reason): %s", err)
	}

	if b.Fullname() != name {
		t.Fatalf("b.Fullname() (%#v) doesn't match our input value", err)
	}

	if b.MARCOrgCode != "oru" {
		t.Fatalf(`b.MARCOrgCode (%#v) should have been "oru"`, b.MARCOrgCode)
	}
	if b.Keyword != "courage_3" {
		t.Fatalf(`b.Keyword (%#v) should have been "courage_3"`, b.Keyword)
	}

	if b.Version != 1 {
		t.Fatalf("Batch %#v: version wasn't 1", b)
	}
}

func TestTSV(t *testing.T) {
	var err error
	var workDir, testDir string
	var infos []os.FileInfo

	var title = &Title{
		LCCN:               "sn12345678",
		Name:               "Treehugger's Digest",
		PlaceOfPublication: "Eugene, Oregon",
		Location:           "somewhereOnDisk",
	}

	workDir, err = os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current directory: %s", err)
	}
	testDir = filepath.Join(workDir, "../cmd/server")
	infos, err = fileutil.ReaddirSorted(testDir)
	if err != nil {
		t.Fatalf("Error reading %q: %s", testDir, err)
	}

	var i = &Issue{
		MARCOrgCode: "oru",
		RawDate:     "2001-02-03",
		Edition:     4,
		Location:    "/mnt/news/data/workflow/2004260523-2001020304-1",
	}
	title.AddIssue(i)
	for _, file := range fileutil.InfosToFiles(infos) {
		var loc = filepath.Join(i.Location, file.Name)
		i.Files = append(i.Files, &File{File: file, Issue: i, Location: loc})
	}

	var expectedTSV = strings.Join([]string{
		"nil",
		"somewhereOnDisk\\tsn12345678\\tTreehugger's Digest\\tEugene, Oregon\\t000001",
		"/mnt/news/data/workflow/2004260523-2001020304-1",
		"2001020304", "",
		"internal,main.go,middleware.go,migrate_issue_metadata_entry.go",
	}, "\t")
	var tsv = i.TSV()
	if tsv != expectedTSV {
		t.Errorf("Issue's TSV should have been %q, but was %q", expectedTSV, tsv)
	}
}
