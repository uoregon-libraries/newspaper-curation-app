package jobs

import (
	"path/filepath"

	"github.com/uoregon-libraries/gopkg/bagit"
	"github.com/uoregon-libraries/gopkg/fileutil/manifest"
	"github.com/uoregon-libraries/gopkg/hasher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
)

// WriteBagitManifest runs our bag tag-file generator
type WriteBagitManifest struct {
	*BatchJob
}

// shaLookup is a bagit "cache" which we use to look for pre-computed SHA
// information to avoid re-SHA-ing files that don't need it
type shaLookup struct {
	baseDir string
	Hits    int64
	Misses  int64
}

// GetSum checks if there's a manifest in the given path, and uses it to look
// for a sum if present
func (l *shaLookup) GetSum(path string) (string, bool) {
	var fullpath = filepath.Join(l.baseDir, path)
	var dir, fname = filepath.Split(fullpath)
	var m, err = manifest.Open(dir)
	if err != nil || m.Hasher == nil {
		return "", false
	}

	// Out of paranoia we make sure the hasher is definitely SHA256
	if m.Hasher.Name != hasher.SHA256 {
		return "", false
	}

	for _, f := range m.Files {
		if fname == f.Name {
			l.Hits++
			return f.Sum, true
		}
	}

	l.Misses++
	return "", false
}

// SetSum does nothing - this isn't a proper cache, just a way to shortcut
// files that already had sums calculated
func (l *shaLookup) SetSum(_, _ string) {
}

// Process implements Processor, writing out the data manifest, bagit.txt, and
// the tag manifest
func (j *WriteBagitManifest) Process(*config.Config) ProcessResponse {
	j.Logger.Debugf("Writing bag manifest")
	var b = bagit.New(j.DBBatch.Location, hasher.NewSHA256())
	var cache = &shaLookup{baseDir: j.DBBatch.Location}
	b.Cache = cache
	var err = b.WriteTagFiles()
	if err != nil {
		j.Logger.Errorf("Unable to write bagit tag files for %q: %s", j.DBBatch.Location, err)
		return PRFailure
	}

	j.Logger.Debugf("Done writing bag manifest: %d cache hits, %d cache misses", cache.Hits, cache.Misses)
	return PRSuccess
}

// ValidateTagManifest verifies that the tagmanifest file accurately represents
// the bagit "tag" files. It's basically here to validate the the bagit work is
// done without the cost of verifying the full manifest.
type ValidateTagManifest struct {
	*BatchJob
}

// Process implements Processor, verifying the tag manifest
func (j *ValidateTagManifest) Process(*config.Config) ProcessResponse {
	var b = bagit.New(j.DBBatch.Location, hasher.NewSHA256())
	var err = b.ReadManifests()
	if err != nil {
		j.Logger.Errorf("Unable to read bagit manifests for %q: %s", j.DBBatch.Location, err)
		return PRFailure
	}

	err = b.GenerateTagSums()
	if err != nil {
		j.Logger.Errorf("Unable to generate bagit tag manifest for %q: %s", j.DBBatch.Location, err)
		return PRFailure
	}

	var discrepancies = bagit.Compare("tag manifest", b.ManifestTagSums, b.ActualTagSums)

	// An invalid manifest *should* mean the bagit process is incomplete, so we
	// don't log this as a HOLY WTF error
	if len(discrepancies) > 0 {
		j.Logger.Warnf("Checksums didn't match, failing job...")
		for _, s := range discrepancies {
			j.Logger.Debugf("- %s", s)
		}
		return PRFailure
	}

	return PRSuccess
}
