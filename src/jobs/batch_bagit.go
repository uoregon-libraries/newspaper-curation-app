package jobs

import (
	"github.com/uoregon-libraries/gopkg/bagit"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
)

// WriteBagitManifest runs our bag tag-file generator
type WriteBagitManifest struct {
	*BatchJob
}

// Process implements Processor, writing out the data manifest, bagit.txt, and
// the tag manifest
func (j *WriteBagitManifest) Process(*config.Config) bool {
	var b = bagit.New(j.DBBatch.Location)
	var err = b.WriteTagFiles()
	if err != nil {
		j.Logger.Errorf("Unable to write bagit tag files for %q: %s", j.DBBatch.Location, err)
		return false
	}

	return true
}

// ValidateTagManifest verifies that the tagmanifest file accurately represents
// the bagit "tag" files. It's basically here to validate the the bagit work is
// done without the cost of verifying the full manifest.
type ValidateTagManifest struct {
	*BatchJob
}

// Process implements Processor, verifying the tag manifest
func (j *ValidateTagManifest) Process(*config.Config) bool {
	var a, b = bagit.New(j.DBBatch.Location), bagit.New(j.DBBatch.Location)
	var err = a.ReadManifests()
	if err != nil {
		j.Logger.Errorf("Unable to read bagit manifests for %q: %s", j.DBBatch.Location, err)
		return false
	}

	err = b.GenerateTagSums()
	if err != nil {
		j.Logger.Errorf("Unable to generate bagit tag manifest for %q: %s", j.DBBatch.Location, err)
		return false
	}

	// An invalid manifest *should* mean the bagit process is incomplete, so we
	// don't log this as a HOLY WTF error
	if len(a.Checksums) != len(b.Checksums) {
		j.Logger.Warnf("Checksums didn't match, failing job...")
		return false
	}
	for i := range a.Checksums {
		if *a.Checksums[i] != *b.Checksums[i] {
			j.Logger.Warnf("Checksums didn't match, failing job...")
			return false
		}
	}

	return true
}
