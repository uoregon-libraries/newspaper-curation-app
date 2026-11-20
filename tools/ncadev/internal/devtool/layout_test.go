package devtool

import (
	"slices"
	"testing"
)

// sampleConfig is trimmed "podman compose config" output
const sampleConfig = `
services:
  db:
    logging:
      driver: k8s-file
      options:
        path: /nca/logs/db.log
  web:
    logging:
      driver: k8s-file
      options:
        path: /nca/logs/web.log
  cli:
    image: foo
volumes:
  batches:
    driver: local
    driver_opts:
      device: /nca/vols/news/production-batches
      o: bind
      type: none
  db: {}
  go-cache: {}
  issue-cache:
    driver: local
    driver_opts:
      device: /nca/vols/cache
      o: bind
      type: none
  news:
    driver: local
    driver_opts:
      device: /nca/vols/news/
      o: bind
      type: none
  tmpfs-thing:
    driver: local
    driver_opts:
      device: tmpfs
      o: size=100m
      type: tmpfs
`

func TestParseLayout(t *testing.T) {
	var l, err = parseLayout([]byte(sampleConfig))
	if err != nil {
		t.Fatalf("parseLayout: %s", err)
	}

	var wantBinds = map[string]string{
		"batches":     "/nca/vols/news/production-batches",
		"issue-cache": "/nca/vols/cache",
		"news":        "/nca/vols/news",
	}
	if len(l.binds) != len(wantBinds) {
		t.Errorf("binds = %v, want %v", l.binds, wantBinds)
	}
	for k, v := range wantBinds {
		if l.binds[k] != v {
			t.Errorf("binds[%q] = %q, want %q", k, l.binds[k], v)
		}
	}

	if want := []string{"/nca/logs"}; !slices.Equal(l.logDirs, want) {
		t.Errorf("logDirs = %q, want %q", l.logDirs, want)
	}

	// The cache is skipped, and batches is inside news
	if got, want := l.savedBinds(), []string{"news"}; !slices.Equal(got, want) {
		t.Errorf("savedBinds() = %q, want %q", got, want)
	}

	var dirs = l.hostDirs()
	for _, want := range []string{"/nca/vols/cache", "/nca/vols/news/sftp", "/nca/vols/news/production-batches", "/nca/logs"} {
		if !slices.Contains(dirs, want) {
			t.Errorf("hostDirs() = %q, missing %q", dirs, want)
		}
	}
}

func TestParseLayoutNoBinds(t *testing.T) {
	var l, err = parseLayout([]byte("volumes:\n  news: {}\n"))
	if err != nil {
		t.Fatalf("parseLayout: %s", err)
	}
	if len(l.binds) != 0 || len(l.savedBinds()) != 0 || len(l.hostDirs()) != 0 {
		t.Errorf("expected an empty layout, got %#v", l)
	}
}
