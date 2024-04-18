package jobs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

type fdata struct {
	path     string
	contents string
	sum      string
}

// files is our list of files, their contents, and expected SHA256 sums for
// making a dummy bag
var files = []fdata{
	{path: "foo.txt", contents: "I am a lonely little text file, friends.", sum: ""},
	{path: "bar.txt", contents: "Me, too!", sum: ""},
	{path: "subdir/other.txt", contents: "I live underneath the other files. It's nice down here.", sum: ""},
}

func getdir(t *testing.T) (tmpdir string) {
	// Create a temporary directory for the test
	var err error
	tmpdir, err = os.MkdirTemp("", "test-bagit-")
	if err != nil {
		t.Fatalf("Failed to create temporary directory: %s", err)
	}
	var bagdir = filepath.Join(tmpdir, "data")
	err = os.Mkdir(bagdir, 0700)
	if err != nil {
		t.Fatalf(`Failed to create "data" directory %q: %s`, bagdir, err)
	}

	// Add some dummy files to the data dir
	for _, f := range files {
		var fullpath = filepath.Join(bagdir, f.path)
		var dir = filepath.Dir(fullpath)
		err = os.MkdirAll(dir, 0700)
		if err != nil {
			t.Fatalf("Unable to create directory %q: %s", dir, err)
		}
		err = os.WriteFile(fullpath, []byte(f.contents), 0600)
		if err != nil {
			t.Fatalf("Unable to create dummy file %q: %s", f.path, err)
		}
	}

	return tmpdir
}

// getBatchJob creates our dummy BatchJob. Yes, this reeks of horrible dependency
// pain and should really be refactored.
func getBatchJob(pth string) *BatchJob {
	var batch = &models.Batch{Location: pth}
	var dbJob = &models.Job{}
	var batchJob = &BatchJob{Job: NewJob(dbJob), DBBatch: batch}

	// The logger has to be custom, as our built-in job logger logs to the
	// database. Good in production, terrible in testing.
	batchJob.Logger = logger.New(logger.Debug, false)

	return batchJob
}

func TestWriteBagitManifest(t *testing.T) {
	var tmpdir = getdir(t)
	defer os.RemoveAll(tmpdir)

	var batchJob = getBatchJob(tmpdir)
	var j = &WriteBagitManifest{BatchJob: batchJob}

	var resp = j.Process(&config.Config{})
	if resp != PRSuccess {
		t.Errorf("Expected PRSuccess, got %v", resp)
	}

	// Check if the tag files were created
	var tagFiles = []string{"bagit.txt", "manifest-sha256.txt", "tagmanifest-sha256.txt"}
	for _, file := range tagFiles {
		var fullpath = filepath.Join(tmpdir, file)
		if !fileutil.Exists(fullpath) {
			t.Errorf("Expected file %q to exist, but it doesn't", fullpath)
		}
	}

	// Make sure manifest is correct for our dummy files
	var fullpath = filepath.Join(tmpdir, "manifest-sha256.txt")
	var raw, err = os.ReadFile(fullpath)
	if err != nil {
		t.Fatalf("Unable to read %q: %s", fullpath, err)
	}
	var lines = strings.Split(string(raw), "\n")
	var expectedLines = []string{
		"07c9b7c5005442cd3b1ef28028417ffb068a2d9426a3d37fa2b8c12b4e79c7dd  data/bar.txt",
		"ec6e72eb877ee399c6bfd08620bd98432eba09f3d941faf51dbcc283a8a695ad  data/foo.txt",
		"712c0490983ef62ce7fe733a90305e46d42da100df600d5179c2c142acfb7108  data/subdir/other.txt",
		"",
	}
	var expected = len(expectedLines)
	var got = len(lines)
	if got != expected {
		t.Errorf("manifest had %d lines; it should have had %d", got, expected)
	}
	for i, got := range lines {
		var expected = expectedLines[i]
		if got != expected {
			t.Errorf("manifest line %d should have been %q, got %q", i, expected, got)
		}
	}
}

func TestValidateTagManifest(t *testing.T) {
	// First run a bag writer job
	var tmpdir = getdir(t)
	defer os.RemoveAll(tmpdir)
	var batchJob = getBatchJob(tmpdir)
	var writeJob = &WriteBagitManifest{BatchJob: batchJob}

	var resp = writeJob.Process(&config.Config{})
	if resp != PRSuccess {
		t.Errorf("Expected PRSuccess, got %v", resp)
	}

	// Now create a validator job
	var validateJob = &ValidateTagManifest{BatchJob: batchJob}
	resp = validateJob.Process(&config.Config{})
	if resp != PRSuccess {
		t.Errorf("Expected PRSuccess, got %v", resp)
	}

	// Modify the manifest to simulate a discrepancy, then re-run the validator
	var pth = filepath.Join(tmpdir, "manifest-sha256.txt")
	var err = os.WriteFile(pth, []byte("CHANGED HAH!"), 0600)
	if err != nil {
		t.Fatalf("Failed to modify test file %q: %s", pth, err)
	}

	// Run the process again
	resp = validateJob.Process(&config.Config{})
	if resp != PRFailure {
		t.Errorf("Expected PRFailure, got %v", resp)
	}
}
