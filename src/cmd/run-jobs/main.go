// This script traverses the database to find born-digital issues in
// WORKFLOW_PATH which need to be split, then splits them (one PDF per page),
// converts pages to PDF/A, and moves them to PDF_PAGE_REVIEW_PATH for
// reordering / cleanup.  The intact originals are then backed up.

package main

import (
	"config"
	"db"
	"fmt"
	"jobs"
	"logger"
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
var titles = make(map[string]*schema.Title)

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
		logger.Fatal("Invalid configuration: %s", err)
	}

	err = db.Connect(c.DatabaseConnect)
	if err != nil {
		logger.Fatal("Unable to connect to the database: %s", err)
	}

	return c
}

func main() {
	var c = getOpts()
	var err = db.LoadTitles()
	if err != nil {
		logger.Fatal("Cannot load titles: %s", err)
	}
	logger.Debug("Looking for SFTP issues to move")
	for _, job := range jobs.FindPendingSFTPIssueMoverJobs() {
		job.Process(c)
	}
	logger.Debug("Looking for page split jobs to process")
	for _, job := range jobs.FindPendingPageSplitJobs() {
		job.Process(c)
	}
	logger.Debug("Complete")
}
