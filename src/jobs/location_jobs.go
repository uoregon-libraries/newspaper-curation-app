package jobs

import "github.com/uoregon-libraries/newspaper-curation-app/src/config"

// SetBatchLocation is a simple job to update a batch location after files are
// copied or movied somewhere
type SetBatchLocation struct {
	*BatchJob
}

// Process just updates the batch's location field
func (j *SetBatchLocation) Process(c *config.Config) bool {
	j.DBBatch.Location = j.db.Args[locArg]
	var err = j.DBBatch.Save()
	if err != nil {
		j.Logger.Errorf("Error setting batch.location for id %d: %s", j.DBBatch.ID, err)
		return false
	}

	return true
}
