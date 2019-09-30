package main

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

const csi = "\033["
const ansiReset = csi + "0m"
const ansiIntenseRed = csi + "31;1m"

// Command-line options
type _opts struct {
	cli.BaseOptions
	Live bool `long:"live" description:"Run the issue deletion operation rather than just showing what would be done"`
}

var opts _opts
var database *sql.DB

func getConfig() {
	var c = cli.New(&opts)
	c.AppendUsage("Deletes all issues from the workflow locations if they're " +
		"part of a batch which has been flagged as being completely done.")

	var conf = c.GetConf()
	var err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}
}

func main() {
	getConfig()

	var batches, err = db.FindLiveArchivedBatches()

	if err != nil {
		logger.Fatalf("Unable to query for batches needing to be closed out: %s", err)
	}

	var op = db.DB.Operation()
	op.Dbg = db.Debug
	op.BeginTransaction()

	if opts.Live {
		warning(batches)
	}

	for _, b := range batches {
		logger.Infof("Closing batch %q", b.FullName())
		b.Close()
	}

	err = purgeIssues()
	if err != nil {
		logger.Fatalf("Unable to purge issue directories: %s", err)
	}
}

func warning(batches []*db.Batch) {
	fmt.Println(ansiIntenseRed + "Warning!" + ansiReset + "  The following batches " +
		"will be closed out, and their issues " + ansiIntenseRed +
		"permanently removed from local disk" + ansiReset + ":")
	for _, b := range batches {
		fmt.Printf("- %q\n", b.FullName())
	}

	fmt.Println()
	for i := 15; i > 0; i-- {
		if i%5 == 0 {
			fmt.Printf("You have %d seconds to cancel this operation (CTRL+C)\n", i)
		}
		time.Sleep(time.Second)
	}
}

func purgeIssues() error {
	var issues, err = db.FindCompletedIssuesReadyForRemoval()
	if err != nil {
		return fmt.Errorf("error looking for issues in live_done batches: %s", err)
	}

	for _, issue := range issues {
		if !opts.Live {
			fmt.Printf("(DRY RUN) Would remove %q\n", issue.Location)
			continue
		}

		fmt.Printf("Removing %q\n", issue.Location)
		err = os.RemoveAll(issue.Location)
		if err != nil {
			return fmt.Errorf("unable to remove issue location %q: %s", issue.Location, err)
		}
		issue.Location = ""
		err = issue.Save()
		if err != nil {
			return fmt.Errorf("unable to remove issue %d's location (%s) from database: %s", issue.ID, issue.Location, err)
		}
	}

	return nil
}
