// This script forcibly completes jobs that are failed or pending so that the
// next job in the sequence can begin. This is only useful for jobs that are
// stuck in some way, such as a directory rename where the source is missing
// and the destination isn't actually desired. This is rare, but can happen if
// jobs fail catastrophically due to files manually being removed.
//
// Please don't use this script except in exceedingly rare situations. Most of
// the time it will have to be run, then a one-queue worker must be run and
// monitored, and subsequent jobs may or may not succeed, resulting in more
// runs of this tool.

package main

import (
	"crypto/md5"
	"fmt"
	"log/slog"
	"os"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

func getOpts() *cli.CLI {
	var opts cli.BaseOptions
	var c = cli.New(&opts)
	var conf = c.GetConf()

	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		slog.Error("Unable to connect to the database", "error", err)
	}

	// This is how we prevent people from doing this without *really* thinking it
	// through first: you need to specify a status, a queue string AND THEN the md5sum of
	// those two combined.
	if len(c.Args) < 3 {
		c.UsageFail("Error: you must specify a status, queue name, and a password")
	}

	return c
}

// validQueueName copies in the list of valid queues for easier validation
func validQueueName(name string) bool {
	for _, jType := range models.ValidJobTypes {
		var jt = string(jType)
		if jt == name {
			return true
		}
	}
	return false
}

func main() {
	var c = getOpts()
	var status = c.Args[0]
	if status != "pending" && status != "failed" {
		c.UsageFail("Error: invalid status %q", status)
	}

	var queue = c.Args[1]
	if !validQueueName(queue) {
		c.UsageFail("Error: invalid queue name %q", queue)
	}

	var expected = fmt.Sprintf("%x", md5.Sum([]byte(status+queue)))
	if c.Args[2] != expected {
		c.UsageFail("Error: incorrect password")
	}

	var jobs, err = models.FindJobsByStatusAndType(models.JobStatus(status), models.JobType(queue))
	if err != nil {
		slog.Error("Can't get next job from the database", "status", status, "queue", queue, "error", err)
		os.Exit(1)
	}

	for _, j := range jobs {
		err = models.CompleteJob(j)
		if err != nil {
			slog.Error("Unable to complete job", "job id", j.ID, "error", err)
		}

		slog.Info("Closed job", "job id", j.ID)
	}
}
