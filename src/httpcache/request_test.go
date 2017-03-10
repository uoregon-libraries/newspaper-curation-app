package httpcache

import (
	"testing"
)

func assertAutoRequest(t *testing.T, url, subdir, expectedFilename, expectedExt string) {
	var r = AutoRequest(url, subdir)

	if r.URL != url {
		t.Fatalf("Request (%#v) had the wrong URL; expected %s", r, url)
	}
	if r.Subdirectory != subdir {
		t.Fatalf("Request (%#v) had the wrong Subdirectory; expected %s", r, subdir)
	}
	if r.Filename != expectedFilename {
		t.Fatalf("Request (%#v) had the wrong Filename; expected %s", r, expectedFilename)
	}
	if r.Extension != expectedExt {
		t.Fatalf("Request (%#v) had the wrong Extension; expected %s", r, expectedExt)
	}
}

func TestAutoRequest(t *testing.T) {
	assertAutoRequest(t, "http://oregonnews.uoregon.edu/batches/batch_foo_bar.json",
		"subdir", "batch_foo_bar-5AVNKD7E.json", "json")
	assertAutoRequest(t, "http://oregonnews.uoregon.edu/batches/", "subdir", "batches-QSMI620P", "")
	assertAutoRequest(t, "", "subdir", "index-7J52DCBI.html", "html")
	assertAutoRequest(t, "/", "subdir", "index-KP0FTT5O.html", "html")
	assertAutoRequest(t, "http://oregonnews.uoregon.edu", "subdir", "index-BUBGO14U.html", "html")
}
