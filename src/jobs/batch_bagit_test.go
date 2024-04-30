package jobs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/fileutil/manifest"
	"github.com/uoregon-libraries/gopkg/hasher"
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
	{path: "baz.txt", contents: "Why lonely? There are a lot of us now.", sum: ""},
	{path: "quux.txt", contents: "hhhhhhhhhhhhhhhhhhhhhhhhhh...", sum: ""},
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

	// Write manifest files with fake precomputed SHAs
	var m = manifest.New(filepath.Join(tmpdir, "data"))
	m.Hasher = hasher.NewSHA256()
	m.Files = []manifest.FileInfo{
		{Name: "nonexistent.txt", Sum: "this entry won't show up anywhere"},
		{Name: "foo.txt", Sum: "not gonna calculate me!"},
		{Name: "quux.txt", Sum: "123abc"},
	}
	var err = m.Write()
	if err != nil {
		t.Fatalf("Unable to write manifest file 1: %s", err)
	}

	m = manifest.New(filepath.Join(tmpdir, "data", "subdir"))
	m.Hasher = hasher.NewSHA256()
	m.Files = []manifest.FileInfo{
		{Name: "other.txt", Sum: "invalid-sum"},
	}
	err = m.Write()
	if err != nil {
		t.Fatalf("Unable to write manifest file 2: %s", err)
	}

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

	// Make sure manifest used the precomputed SHA sums when available, and real
	// sums when the manifest didn't provide any
	var fullpath = filepath.Join(tmpdir, "manifest-sha256.txt")
	var raw []byte
	raw, err = os.ReadFile(fullpath)
	if err != nil {
		t.Fatalf("Unable to read %q: %s", fullpath, err)
	}
	var lines = strings.Split(string(raw), "\n")
	var expectedLines = []string{
		"<ignore>  data/.manifest",
		"07c9b7c5005442cd3b1ef28028417ffb068a2d9426a3d37fa2b8c12b4e79c7dd  data/bar.txt",
		"6e4afe213d8cb12b9cb188e34ab0be5da9cafc6b3498b61aefa1e1fa51d84af7  data/baz.txt",
		"not gonna calculate me!  data/foo.txt",
		"123abc  data/quux.txt",
		"<ignore>  data/subdir/.manifest",
		"invalid-sum  data/subdir/other.txt",
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
			// We have to ignore the .manifest lines because that's not part of the
			// test, and manifest files change when created in order to record the
			// current time
			if strings.HasSuffix(expected, "/.manifest") {
				continue
			}
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
