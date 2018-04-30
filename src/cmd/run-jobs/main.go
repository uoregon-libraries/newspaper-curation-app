// This script finds and runs pending jobs, scans for page review issues which
// have been renamed and are ready for derivatives, and will eventually perform
// all automated processes Batch Maker has to offer.

package main

import (
	"config"
	"db"
	"fmt"
	"jobs"
	"os"
	"schema"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/uoregon-libraries/gopkg/interrupts"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/wordutils"
)

var runners struct {
	sync.Mutex
	list []*jobs.Runner
}

var isDone int32

func addRunner(r *jobs.Runner) {
	runners.Lock()
	runners.list = append(runners.list, r)
	runners.Unlock()
}

func quit() {
	atomic.StoreInt32(&isDone, 1)
	runners.Lock()
	for _, r := range runners.list {
		r.Stop()
	}
	runners.Unlock()
}

func done() bool {
	return atomic.LoadInt32(&isDone) == 1
}

// Command-line options
var opts struct {
	ConfigFile string `short:"c" long:"config" description:"path to NCA config file" required:"true"`
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
	wrapBullet("* watchall: Runs watchers for all queues and the page review " +
		"issues in a relatively sane configuration.  Use this unless you need the " +
		`more complex granularity offered by "watch" and "watch-page-review"`)
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
		logger.Fatalf("Invalid configuration: %s", err)
	}

	err = db.Connect(c.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Unable to connect to the database: %s", err)
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

	// On CTRL-C / kill, try to finish the current task before exiting
	interrupts.TrapIntTerm(quit)

	var action string
	action, args = args[0], args[1:]
	switch action {
	case "requeue":
		requeue(args)
	case "watch":
		watch(c, args...)
	case "watch-page-review":
		watchPageReview(c)
	case "watchall":
		runAllQueues(c)
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
		logger.Errorf("Invalid job id %q", idString)
		return
	}

	var j = jobs.Find(id)
	if j == nil {
		logger.Errorf("Cannot requeue job id %d: no such job", id)
		return
	}
	var failStatus = jobs.JobStatusFailed
	if j.Status != string(failStatus) {
		logger.Errorf("Cannot requeue job id %d: status is %s (it must be %s to requeue)", id, j.Status, failStatus)
		return
	}

	logger.Infof("Requeuing job %d", j.ID)
	var err = j.Requeue()
	if err != nil {
		logger.Errorf("Unable to requeue job %d: %s", j.ID, err)
	}
}

func validateJobQueue(queue string) {
	if !validQueues[queue] {
		usageFail("Invalid job queue %q", queue)
	}
}

func watch(c *config.Config, queues ...string) {
	if len(queues) == 0 {
		usageFail("Error: you must specify one or more queues to watch")
	}

	var jobTypes = make([]jobs.JobType, len(queues))
	for i, queue := range queues {
		validateJobQueue(queue)
		jobTypes[i] = jobs.JobType(queue)
	}
	watchJobTypes(c, jobTypes...)
}

func watchJobTypes(c *config.Config, jobTypes ...jobs.JobType) {
	var r = jobs.NewRunner(c, jobTypes...)
	addRunner(r)
	r.Watch(time.Second * 10)
}

func watchPageReview(c *config.Config) {
	logger.Infof("Watching page review folders")

	var nextAttempt time.Time
	for !done() {
		if time.Now().After(nextAttempt) {
			scanPageReviewIssues(c)
			nextAttempt = time.Now().Add(10 * time.Minute)
		}

		// Try not to eat all the CPU
		time.Sleep(time.Second)
	}
}

// runAllQueues fires up multiple goroutines to watch all the queues in a
// fairly sane way so that important processes like moving SFTP issues can
// happen quickly, while CPU-bound processes won't fight each other.
func runAllQueues(c *config.Config) {
	waitFor(
		func() { watchPageReview(c) },
		func() {
			// Potentially slow filesystem moves
			watchJobTypes(c,
				jobs.JobTypeMoveIssueToWorkflow,
				jobs.JobTypeMoveIssueToPageReview,
				jobs.JobTypeMoveMasterFiles,
			)
		},
		func() {
			// Slow jobs: expensive process spawning or file crunching
			watchJobTypes(c,
				jobs.JobTypePageSplit,
				jobs.JobTypeMakeDerivatives,
				jobs.JobTypeWriteBagitManifest,
			)
		},
		func() {
			// Fast jobs: file renaming, hard-linking, running templates for very
			// simple XML output, etc.
			watchJobTypes(c,
				jobs.JobTypeBuildMETS,
				jobs.JobTypeCreateBatchStructure,
				jobs.JobTypeMakeBatchXML,
				jobs.JobTypeMoveBatchToReadyLocation,
			)
		},
	)
}

// waitFor runs all the passed-in functions concurrently and returns when
// they're all complete
func waitFor(fns ...func()) {
	var wg sync.WaitGroup

	for _, fn1 := range fns {
		wg.Add(1)
		go func(fn2 func()) {
			fn2()
			wg.Done()
		}(fn1)
	}

	wg.Wait()
}
