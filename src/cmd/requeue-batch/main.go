package main

import (
	"fmt"
	"os"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
)

// Command-line options
type _opts struct {
	cli.BaseOptions
	BatchID int `long:"batch-id" description:"The id of the batch which needs to be re-queued" required:"true"`
}

var opts _opts

func getBatch() *db.Batch {
	var c = cli.New(&opts)
	c.AppendUsage("Re-queues a specified batch without regard for its " +
		"current state.  This should only be done when a batch needs to be " +
		"manually fixed, and only after issues that need a fix have already " +
		"been pulled off the batch by manually setting their batch_id to 0 in " +
		"the database, updating their workflow status, and deleting their METS " +
		"XML file.  If any of this sounds confusing or scary, you do NOT want " +
		"to run this tool.")
	c.AppendUsage(fmt.Sprintf("For safety, this only works on batches whose "+
		"status is %s.  This again must be manually set, but at least ensures a "+
		"typoed batch ID doesn't cause major problems unless you make the same "+
		"typo twice.", db.BatchStatusFailedQC))

	var conf = c.GetConf()
	var err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}
	var batch *db.Batch
	batch, err = db.FindBatch(opts.BatchID)
	if err != nil {
		logger.Fatalf("Error trying to look up batch: %s", err)
	}
	if batch == nil {
		logger.Fatalf("Batch %d wasn't found in the database", opts.BatchID)
	}
	if batch.Status != db.BatchStatusFailedQC {
		logger.Fatalf("Batch %d hasn't failed QC, and is ineligible for re-queueing", opts.BatchID)
	}

	return batch
}

func main() {
	var batch = getBatch()

	// Give us some time to change our minds even after the checks above
	for i := 10; i > 0; i-- {
		logger.Warnf("Re-queueing %q in %d seconds", batch.FullName(), i)
		time.Sleep(time.Second)
	}

	// Remove the entire batch - these are all bagit files or hard-links, so
	// we're not losing anything that's not easy to replace
	var err = os.RemoveAll(batch.Location)
	if err != nil {
		logger.Fatalf("Unable to remove batch files from %q: %s", batch.Location, err)
	}

	// Flag the batch as pending again to avoid confusion
	batch.Status = db.BatchStatusPending
	err = batch.Save()
	if err != nil {
		logger.Fatalf("Unable to update batch status to 'pending' - operation aborted")
	}

	// Finally: requeue
	err = jobs.QueueMakeBatch(batch)
	if err != nil {
		logger.Fatalf("Error queueing batch regeneration: %s", err)
	}
}
