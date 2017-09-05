// This script traverses the database to find born-digital issues in
// WORKFLOW_PATH which need to be split, then splits them (one PDF per page),
// converts pages to PDF/A, and moves them to PDF_PAGE_REVIEW_PATH for
// reordering / cleanup.  The intact originals are then backed up.

package main

import (
	"config"
	"db"
	"fmt"
	"log"
	"os"
	"schema"
	"wordutils"

	"github.com/jessevdk/go-flags"
)

// Command-line options
var opts struct {
	ConfigFile string `short:"c" long:"config" description:"path to P2C config file" required:"true"`
}

var p *flags.Parser

// wrap is a helper to wrap a usage message at 80 characters and print a
// newline afterward
func wrap(msg string) {
	fmt.Fprint(os.Stderr, wordutils.Wrap(msg, 80))
	fmt.Fprintln(os.Stderr)
}

func usageFail(format string, args ...interface{}) {
	wrap(fmt.Sprintf(format, args...))
	fmt.Fprintln(os.Stderr)
	p.WriteHelp(os.Stderr)
	os.Exit(1)
}

func getOpts() *config.Config {
	p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	var _, err = p.Parse()

	if err != nil {
		usageFail("Error: %s", err)
	}

	var c *config.Config
	c, err = config.Parse(opts.ConfigFile)
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	err = db.Connect(c.DatabaseConnect)
	if err != nil {
		log.Fatalf("Error trying to connect to database: %s", err)
	}

	return c
}

func main() {
	var c = getOpts()
	for _, issue := range getIssuesAwaitingSplit() {
		issue.ProcessPDFs(c)
	}
}

// getIssuesAwaitingSplit finds all issues in the database which are awaiting
// PDF processing
func getIssuesAwaitingSplit() []*Issue {
	var dbIssues, err = db.FindAllAwaitingPDFProcessing()
	if err != nil {
		log.Fatalf("ERROR - Unable to find issues needing processing in the database: %s", err)
	}

	var issues = make([]*Issue, len(dbIssues))
	for i, dbi := range dbIssues {
		if dbi.Error != "" {
			log.Printf("WARN - Skipping issue (id %d, location %q): %s", dbi.ID, dbi.Location, dbi.Error)
			continue
		}
		issues[i] = &Issue{DBIssue: dbi, Issue: &schema.Issue{Location: dbi.Location}}
	}

	return issues
}
