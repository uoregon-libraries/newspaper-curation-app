package jobs

import (
	"path/filepath"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/derivatives/batchxml"
)

// MakeBatchXML wraps a BatchJob and implements Processor to create the
// top-level batch XML file
type MakeBatchXML struct {
	*BatchJob
}

// Process generates the batch XML file
func (j *MakeBatchXML) Process(c *config.Config) bool {
	var bName = j.DBBatch.FullName()
	j.Logger.Debugf("Generating batch XML for batch %q", bName)

	// Set up variables
	var templatePath = c.BatchXMLTemplatePath
	var outputXMLPath = filepath.Join(j.db.Location, "data", "batch.xml")

	var issues, err = j.DBBatch.Issues()
	if err != nil {
		j.Logger.Errorf("Unable to look up issues for batch id %d (%q): %s", j.DBBatch.ID, bName, err)
		return false
	}

	err = batchxml.New(templatePath, outputXMLPath, j.DBBatch, issues).Transform()
	if err != nil {
		j.Logger.Errorf("Unable to generate Batch XML for batch %d: %s", j.DBBatch.ID, err)
		return false
	}

	j.Logger.Debugf("Batch XML generated")
	return true
}
