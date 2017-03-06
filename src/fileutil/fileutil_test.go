package fileutil

import (
	"testing"
)

// TestFind verifies that Find ... doesn't crash.  This needs a mock for the
// Readdir wrapper function so we can get actual high-level testing without
// relying on a completely unknown filesystem....
func TestFind(t *testing.T) {
	var _, err = Find("/", 2)
	if err != nil {
		t.Fatalf("Got an error trying to read the filesystem!  %s", err)
	}
}
