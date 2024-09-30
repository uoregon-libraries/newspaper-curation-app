package main

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

var opts cli.BaseOptions

var erroredIssuesPath string

func getConfig() {
	var c = cli.New(&opts)
	c.AppendUsage(`Deletes all "stuck" issues couldn't make it into NCA. Issues must have the "AwaitingProcessing" workflow step and at least one dead job ("failed", not "failed_done") to be considered for deletion. They will not be removed if they are tied to a batch or have any pending jobs associated with them. All issues' jobs will be finalized (set to "failed_done") or removed (those that are on hold waiting for the failed job / jobs).`)

	var conf = c.GetConf()
	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}
	erroredIssuesPath = conf.ErroredIssuesPath
}

func main() {
	getConfig()

	logger.Debugf("Scanning for issues to remove")

	var issues, err = models.Issues().InWorkflowStep(schema.WSAwaitingProcessing).Fetch()
	if err != nil {
		logger.Fatalf("Unable to scan database for issues awaiting processing: %s", err)
	}

	remove(issues)

	logger.Debugf("Process complete")
}

func remove(list []*models.Issue) {
	for _, i := range list {
		logger.Debugf("Examining issue id %d (%s)", i.ID, i.HumanName)
		var ok, err = canDelete(i)
		if err != nil {
			logger.Errorf("Error reading data for issue id %d (%s): %s", i.ID, i.HumanName, err)
			return
		}
		if !ok {
			logger.Infof(`Skipping issue id %d (%s): not in a "dead" state`, i.ID, i.HumanName)
			continue
		}

		err = jobs.QueueDeleteStuckIssue(i, erroredIssuesPath)
		if err != nil {
			logger.Errorf("Error queueing issue id %d (%s) for deletion: %s", i.ID, i.HumanName, err)
		}

		logger.Infof("Queued issue id %d (%s) for deletion", i.ID, i.HumanName)
	}
}

// canDelete tells us if an issue is in need of deletion. This is true only if
// all the following are true of the issue:
//
// - It has at least one dead job ("failed", not "failed_done")
// - It is awaiting processing
// - It is not tied to a batch
// - It has no pending jobs still tied to it
func canDelete(i *models.Issue) (bool, error) {
	// We validate there are failed jobs first - everything else should be
	// impossible if there were failed jobs
	var joblist, err = i.Jobs()
	if err != nil {
		return false, err
	}

	var hasFailedJob bool
	for _, j := range joblist {
		switch models.JobStatus(j.Status) {
		case models.JobStatusFailed:
			hasFailedJob = true
		case models.JobStatusOnHold, models.JobStatusSuccessful, models.JobStatusFailedDone:
			continue
		default:
			return false, fmt.Errorf("unexpected data: issue has one or more jobs in a non-purgable status")
		}
	}
	if !hasFailedJob {
		return false, nil
	}

	if i.WorkflowStep != schema.WSAwaitingProcessing {
		return false, fmt.Errorf("unexpected data: issue is not awaiting processing")
	}
	if i.BatchID != 0 {
		return false, fmt.Errorf("unexpected data: issue is tied to a batch")
	}

	return true, nil
}
