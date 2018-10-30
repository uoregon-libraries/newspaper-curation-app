package jobs

import (
	"config"
	"os"
	"path"
)

// MoveBatchToReadyLocation is a very simple job that just renames a batch from
// /path/to/batches/.wip-batch_blah_XXX to /path/to/batches/batch_blah_XXX
type MoveBatchToReadyLocation struct {
	*BatchJob
}

// Process implements Processor by renaming the batch directory
func (j *MoveBatchToReadyLocation) Process(c *config.Config) bool {
	var newPath = path.Join(c.BatchOutputPath, j.DBBatch.FullName())
	var err = os.Rename(j.db.Location, newPath)
	if err != nil {
		j.Logger.Errorf("Unable to rename WIP batch directory (%q -> %q): %s", j.db.Location, newPath, err)
		return false
	}
	j.DBBatch.Location = newPath

	return true
}
