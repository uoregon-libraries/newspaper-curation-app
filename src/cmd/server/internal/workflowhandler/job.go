package workflowhandler

import (
	"jobs"

	"github.com/uoregon-libraries/gopkg/logger"
)

// queueMETSCreation fires off a job to generate an issue's METS XML, logging
// loudly if it fails
func queueMETSCreation(i *Issue) {
	var err = jobs.QueueBuildMETS(i.Issue, i.Location)
	if err != nil {
		logger.Criticalf("Unable to queue METS XML creation for issue id %d: %s", i.ID, err)
	}
}
