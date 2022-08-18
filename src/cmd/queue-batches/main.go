package main

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// Command-line options
type _opts struct {
	cli.BaseOptions
	Redo bool `long:"redo" description:"only queue issues needing a re-batch"`
}

var opts _opts
var titles models.TitleList

func getOpts() *config.Config {
	var c = cli.New(&opts)
	c.AppendUsage("Queues one or more batches depending on the number of " +
		"issues in the database which are flagged as ready for batching.  See " +
		"the MAX_BATCH_SIZE and MIN_BATCH_SIZE settings to control how many " +
		"pages a batch may contain.")
	c.AppendUsage(`If --redo is specified, issues must be in a special "ready for ` +
		`rebatching" state in order to be queued. This is not a state NCA sets ` +
		`normally, and is only needed when there are manual fixes that require ` +
		`hacking the database. In other words, if you don't know what this means, ` +
		`you don't need it.`)
	var conf = c.GetConf()
	var err = dbi.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}

	titles, err = models.Titles()
	if err != nil {
		logger.Fatalf("Unable to find titles in the database: %s", err)
	}

	return conf
}

var conf *config.Config

func main() {
	conf = getOpts()
	logger.Infof("Scanning ready issues for batchability")

	var q = newBatchQueue(conf.MinBatchSize, conf.MaxBatchSize)
	q.FindReadyIssues(opts.Redo)
	for {
		var batch, ok = q.NextBatch()
		if !ok {
			logger.Debugf("No more batches")
			break
		}

		if batch == nil {
			continue
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
		logger.Infof("Sending %q to job runner for creation", batch.Name)
		err = jobs.QueueMakeBatch(batch, conf.BatchOutputPath)
		if err != nil {
			logger.Fatalf("Unable to queue batch %q: %s", batch.Name, err)
		}
	}
}
