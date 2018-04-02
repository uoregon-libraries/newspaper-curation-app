package main

import (
	"cli"
	"config"
	"db"
	"jobs"

	"github.com/uoregon-libraries/gopkg/logger"
)

// Command-line options
type _opts struct {
	cli.BaseOptions
}

var opts _opts

func getOpts() *config.Config {
	var c = cli.New(&opts)
	c.AppendUsage("Queues one or more batches depending on the number of " +
		"issues in the database which are flagged as ready for batching.  See " +
		"the MAX_BATCH_SIZE and MIN_BATCH_SIZE settings to control how many " +
		"pages a batch may contain.")
	var conf = c.GetConf()
	var err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}

	return conf
}

func main() {
	var conf = getOpts()
	logger.Infof("Scanning ready issues for batchability")

	var q = newBatchQueue(conf.MinBatchSize, conf.MaxBatchSize)
	q.FindReadyIssues(conf.EmbargoDays)
	for {
		var batch, ok = q.NextBatch()
		if !ok {
			logger.Debugf("No more batches")
			break
		}

		if batch == nil {
			return
		}

		var issues, err = batch.Issues()
		if err != nil {
			// No idea what this could mean other than maybe an SQL typo
			logger.Fatalf("Unable to pull issues for pending batch: %s", err)
		}

		logger.Infof("Starting a new batch, %q", batch.Name)

		for _, issue := range issues {
			logger.Debugf("Adding %q to batch", issue.Key())
		}

		// Queue the batch
		jobs.QueueMakeBatch(batch)
		logger.Infof("Sending %q to job runner for creation", batch.Name)
	}
}
