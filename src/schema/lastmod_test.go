package schema

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManifestEquivalent(t *testing.T) {
	var f1 = fileInfo{Path: "path1", Size: 1, Checksum: "checksum1"}
	var f2 = fileInfo{Path: "path2", Size: 2, Checksum: "checksum2"}
	var f3 = fileInfo{Path: "path3", Size: 3, Checksum: "checksum3"}
	var f4 = fileInfo{Path: "path4", Size: 4, Checksum: "checksum4"}
	var f5 = fileInfo{Path: "path5", Size: 5, Checksum: "checksum5"}
	var f6 = fileInfo{Path: "path6", Size: 6, Checksum: "checksum6"}
	var a, b = &manifest{}, &manifest{}

	if !a.equiv(b) {
		t.Fatalf("Zero value manifests should be equal")
	}

	a.Path = "/path"
	a.Created = time.Now()
	a.Files = []fileInfo{f1, f2, f3, f4}

	b.Path = a.Path
	b.Created = a.Created
	b.Files = []fileInfo{f1, f2, f3, f4}

	if !a.equiv(b) {
		t.Fatalf("Exact matches should be equivalent")
	}

	a.Files = []fileInfo{f2, f4, f1, f3}
	if !a.equiv(b) {
		t.Fatalf("Order of files shouldn't change equivalence")
	}

	b.Files = append(b.Files, f3)
	if a.equiv(b) {
		t.Fatalf("Dupes should still cause non-equivalence")
	}

	a.Files = []fileInfo{f1, f2, f3, f4, f5}
	b.Files = []fileInfo{f1, f2, f3, f4, f6}

	if a.equiv(b) {
		t.Fatalf("Different file lists shouldn't be equivalent")
	}

	a.Files = b.Files
	a.Path = "/foo"
	b.Path = "/bar"
	if !a.equiv(b) {
		t.Fatalf("Having different paths shouldn't affect equivalence")
	}

	a.Created = time.Now()
	if !a.equiv(b) {
		t.Fatalf("Different create times shouldn't affect equivalence")
	}
}

func _m(t *testing.T) *manifest {
	var cwd, err = os.Getwd()
	if err != nil {
		t.Fatalf("Error getting current directory: %s", err)
		return nil
	}
	var testdata = filepath.Join(cwd, "testdata")
	return newManifest(testdata)
}

func _mkf(name string, size int64, checksum string) fileInfo {
	var cwd, _ = os.Getwd()
	return fileInfo{Path: filepath.Join(cwd, "testdata", name), Size: size, Checksum: checksum}
}

// These are the file manifests for what's in the testdata dir
var expectedFiles = []fileInfo{
	_mkf("a.txt", 30, "df879070"),
	_mkf("b.bin", 5000, "df3b5d6a"),
	_mkf("c.null", 0, "00000000"),
}

func TestManifestBuild(t *testing.T) {
	var m = _m(t)
	var err = m.build()
	if err != nil {
		t.Fatalf("Unable to build manifest: %s", err)
	}

	var expected = len(expectedFiles)
	var got = len(m.Files)
	if expected != got {
		for _, f := range m.Files {
			t.Logf("File: %#v", f)
		}
		t.Fatalf("Invalid manifest: expected to see %d files, but got %d", expected, got)
	}

	m.sortFiles()

	for i := range expectedFiles {
		if m.Files[i] != expectedFiles[i] {
			t.Fatalf("Invalid manifest: expected m.Files[%d] to be %#v, got %#v", i, expectedFiles[i], m.Files[i])
		}
	}
}

func TestManifestWrite(t *testing.T) {
	var m = _m(t)
	m.build()
	var err = m.write()
	if err != nil {
		t.Fatalf("Unable to write manifest: %s", err)
	}
}

func TestManifestRead(t *testing.T) {
	var corpus = _m(t)
	corpus.build()
	corpus.Created = time.Time{}
	var err = corpus.write()
	if err != nil {
		t.Fatalf("Unable to write fake manifest out: %s", err)
	}

	var m = _m(t)
	m.read()

	if !m.Created.IsZero() {
		t.Fatalf("Reading existing manifest didn't result in the expected fake time data")
	}

	var expected = len(expectedFiles)
	var got = len(m.Files)
	if expected != got {
		for _, f := range m.Files {
			t.Logf("File: %#v", f)
		}
		t.Fatalf("Invalid manifest: expected to see %d files, but got %d", expected, got)
	}

	m.sortFiles()

	for i := range expectedFiles {
		if m.Files[i] != expectedFiles[i] {
			t.Fatalf("Invalid manifest: expected m.Files[%d] to be %#v, got %#v", i, expectedFiles[i], m.Files[i])
		}
	}
}
