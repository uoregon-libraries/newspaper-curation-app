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
func (j *WriteBagitManifest) Process(*config.Config) ProcessResponse {
	var b = bagit.New(j.DBBatch.Location)
	var err = b.WriteTagFiles()
	if err != nil {
		j.Logger.Errorf("Unable to write bagit tag files for %q: %s", j.DBBatch.Location, err)
		return PRFailure
	}

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
	var b = bagit.New(j.DBBatch.Location)
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
