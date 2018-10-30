package jobs

import (
	"config"

	"github.com/uoregon-libraries/gopkg/bagit"
)

// WriteBagitManifest runs our bag tag-file generator
type WriteBagitManifest struct {
	*BatchJob
}

// Process implements Processor, writing out the data manifest, bagit.txt, and
// the tag manifest
func (j *WriteBagitManifest) Process(c *config.Config) bool {
	var b = bagit.New(j.db.Location)
	var err = b.WriteTagFiles()
	if err != nil {
		j.Logger.Errorf("Unable to write bagit tag files for %q: %s", j.db.Location, err)
		return false
	}

	return true
}
