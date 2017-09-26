// This script finds and runs pending jobs, scans for page review issues which
// have been renamed and are ready for derivatives, and will eventually perform
// all automated processes Batch Maker has to offer.

package main

import (
	"config"
	"db"
	"fmt"
	"jobs"
	"logger"
	"os"
	"schema"
	"strconv"
	"strings"
	"time"
	"wordutils"

	"github.com/jessevdk/go-flags"
)

// Command-line options
var opts struct {
	ConfigFile string `short:"c" long:"config" description:"path to P2C config file" required:"true"`
}

var p *flags.Parser
var titles = make(map[string]*schema.Title)

var validQueues = make(map[string]bool)
var validQueueList []string

// wrap is a helper to wrap a usage message at 80 characters and print a
// newline afterward
func wrap(msg string) {
	fmt.Fprint(os.Stderr, wordutils.Wrap(msg, 80))
	fmt.Fprintln(os.Stderr)
}

func wrapBullet(msg string) {
	var lines = strings.Split(wordutils.Wrap(msg, 80), "\n")
	for i, line := range lines {
		if i > 0 {
			line = "  " + line
		}
		fmt.Fprint(os.Stderr, line+"\n")
	}
}

func usageFail(format string, args ...interface{}) {
	wrap(fmt.Sprintf(format, args...))
	fmt.Fprintln(os.Stderr)
	p.WriteHelp(os.Stderr)

	fmt.Fprintln(os.Stderr)
	wrap("Valid actions:")
	fmt.Fprintln(os.Stderr)
	wrapBullet("* requeue <job id> [<job id>...]: Creates new jobs by cloning and " +
		`closing the given failed jobs.  Only jobs with a status of "failed" can be requeued.`)
	wrapBullet("* watch <queue name> [<queue name>...]: Watches for jobs in the " +
		"given queue(s), processing them in a loop until CTRL+C is pressed")
	wrapBullet("* watch-page-review: Watches for issues awaiting page review " +
		"(reordering or other manual processing) which are ready to be moved for " +
		"metadata entry.  No job is associated with this action, hence it must run on " +
		"its own, and should only have one copy running at a time.")

	fmt.Fprintln(os.Stderr)
	wrap(fmt.Sprintf("Valid queue names: %s", strings.Join(validQueueList, ", ")))

	os.Exit(1)
}

func getOpts() (*config.Config, []string) {
	p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	var args, err = p.Parse()

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

	return c, args
}

// setupValidQueueNames copies in the list of valid queues for easier validation
func setupValidQueueNames() {
	for _, jType := range jobs.ValidJobTypes {
		var jt = string(jType)
		validQueues[jt] = true
		validQueueList = append(validQueueList, jt)
	}
}

func main() {
	setupValidQueueNames()
	var c, args = getOpts()
	if len(args) < 1 {
		usageFail("Error: you must specify an action")
	}

	var err = db.LoadTitles()
	if err != nil {
		logger.Fatal("Cannot load titles: %s", err)
	}

	var action string
	action, args = args[0], args[1:]
	switch action {
	case "requeue":
		requeue(args)
	case "watch":
		watch(c, args)
	case "watch-page-review":
		watchPageReview(c)
	default:
		usageFail("Error: invalid action")
	}
}

func requeue(ids []string) {
	if len(ids) == 0 {
		usageFail("Error: the requeue action requires at least one job id")
	}

	for _, idString := range ids {
		retryJob(idString)
	}
}

func retryJob(idString string) {
	var id, _ = strconv.Atoi(idString)
	if id == 0 {
		logger.Error("Invalid job id %q", idString)
		return
	}

	var j = jobs.Find(id)
	if j == nil {
		logger.Error("Cannot requeue job id %d: no such job", id)
		return
	}
	var failStatus = jobs.JobStatusFailed
	if j.Status != string(failStatus) {
		logger.Error("Cannot requeue job id %d: status is %s (it must be %s to requeue)", id, j.Status, failStatus)
		return
	}

	logger.Debug("Requeuing job %d", j.ID)
	var err = j.Requeue()
	if err != nil {
		logger.Error("Unable to requeue job %d: %s", j.ID, err)
	}
}

func validateJobQueue(queue string) {
	if !validQueues[queue] {
		usageFail("Invalid job queue %q", queue)
	}
}

func watch(c *config.Config, queues []string) {
	if len(queues) == 0 {
		usageFail("Error: you must specify one or more queues to watch")
	}

	logger.Info("Watching queues: %s", strings.Join(queues, " / "))

	// TODO: Handle CTRL+C

	for _, queue := range queues {
		validateJobQueue(queue)
	}

	var done = false
	for !done {
		var pr = jobs.NextJobProcessor(queues)
		if pr == nil {
			logger.Info("No more jobs; sleeping for 1 minute")
			time.Sleep(time.Minute)
			continue
		}

		logger.Debug("Starting job id %d: %q", pr.JobID(), pr.JobType())
		pr.SetProcessSuccess(pr.Process(c))
		logger.Debug("Finished job id %d", pr.JobID())
	}
}

func watchPageReview(c *config.Config) {
	logger.Info("Watching page review folders")

	// TODO: Handle CTRL+C

	var done = false
	for !done {
		logger.Info("Scanning for page review issues to move")
		scanPageReviewIssues(c)
		logger.Info("Waiting 10 minutes")
		time.Sleep(time.Minute * 10)
	}
}
