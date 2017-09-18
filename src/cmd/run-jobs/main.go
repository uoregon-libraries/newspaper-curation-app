// This script finds and runs pending jobs

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
	ConfigFile      string `short:"c" long:"config" description:"path to P2C config file" required:"true"`
	RetryFailedJobs bool   `long:"retry-failed-jobs" description:"if set, puts failed jobs back into the queue before running pending jobs"`
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

	if opts.RetryFailedJobs {
		retryFailedJobs()
	}
	runPendingJobs(c)
}

func retryFailedJobs() {
	logger.Debug("Looking for failed jobs to requeue")
	for _, j := range jobs.FindAllFailedJobs() {
		logger.Debug("Requeuing job %d", j.ID)
		var err = j.Requeue()
		if err != nil {
			logger.Error("Unable to requeue job %d: %s", j.ID, err)
		}
	}
	logger.Debug("Complete")
}

func runPendingJobs(c *config.Config) {
	logger.Debug("Looking for pending jobs")
	for _, p := range jobs.FindAllPendingJobs() {
		logger.Debug("Starting job id %d: %q", p.JobID(), p.JobType())
		p.SetProcessSuccess(p.Process(c))
		logger.Debug("Finished job id %d", p.JobID())
	}
	logger.Debug("Complete")
}
