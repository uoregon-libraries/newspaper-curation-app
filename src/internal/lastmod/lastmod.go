package lastmod

import (
	"errors"
	"io/fs"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil/manifest"
)

// Time returns the timestamp of when anything in pth was last modified
// *relative to NCA's understanding of it*. The first time a directory is seen,
// it gets a manifest telling us that moment is its modification time. Anytime
// anything major changes (file size, new file, deleted file, or file
// modtimes), NCA records that moment as the new mod time.
//
// File modification times are only looked at to help determine if a change
// occurred - the timestamps could be ancient, set to a future time, or what
// have you, but NCA only tracks that a change has occurred, and the state of
// the files so future changes can be detected.
func Time(pth string) (time.Time, error) {
	// Build a fresh manifest
	var refreshed, err = manifest.Build(pth)
	if err != nil {
		// This can happen when a directory is being moved by another process while
		// this process is scanning it, so we just return time.Now() and ignore the
		// (likely invalid) error
		if errors.Is(err, fs.ErrNotExist) {
			return time.Now(), nil
		}

		return time.Now(), err
	}

	// Open an existing manifest (if one exists)
	var existing *manifest.Manifest
	existing, err = manifest.Open(pth)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return time.Now(), err
	}

	// Different existing manifest (including not having an existing manifest)?
	// Write new data and return the current time.
	if err != nil || !existing.Equiv(refreshed) {
		err = refreshed.Write()
		if err != nil {
			return time.Now(), err
		}
		return refreshed.Created, nil
	}

	// Manifests are the same? Return the existing manifest's creation time.
	return existing.Created, nil
}
