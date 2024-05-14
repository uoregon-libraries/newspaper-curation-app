// This script finds and runs pending jobs, scans for page review issues which
// have been renamed and are ready for derivatives, and will eventually perform
// all automated processes Batch Maker has to offer.

package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uoregon-libraries/gopkg/interrupts"
	ltype "github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/wordutils"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
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
	cli.BaseOptions
	ExitWhenDone bool `long:"exit-when-done" description:"Exit the application when there are no jobs left to run. Note that this may not do anything if running any operations other than the 'watchall' command."`
	Verbose      bool `short:"v" long:"verbose" description:"show verbose debugging when running jobs"`
}

var c *cli.CLI
var titles = make(map[string]*schema.Title)

var validQueues = make(map[string]bool)
var validQueueList []string
var logLevel ltype.LogLevel

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

func getOpts() (*config.Config, []string) {
	c = cli.New(&opts)
	var command = "\033[1;3m"
	var warning = "\033[31;40;1m"
	var reset = "\033[0m"
	c.AppendUsage("Valid actions:")
	c.AppendUsage(command + "requeue" + reset + " <job id> [<job id>...]: Creates new jobs by cloning and " +
		`closing the given failed jobs. Only jobs with a status of "failed" can be requeued.`)
	c.AppendUsage(command + "watchall" + reset + ": Runs watchers for all queues and the page review " +
		"issues in a relatively sane configuration. Use this unless you need the " +
		`more complex granularity offered by "watch" and "watch-page-review"`)
	c.AppendUsage(command + "watch" + reset + " <queue name> [<queue name>...]: Watches for jobs in the " +
		"given queue(s), processing them in a loop until CTRL+C is pressed. " +
		warning + "This usage is not recommended" + reset + " due to the sheer number of queue " +
		"names now in NCA. If this is used, the queue name has to be read from code. " +
		"Consider it a test to prove you're really serious about doing this.")
	c.AppendUsage(command + "run-one" + reset + ": runs a single job and exits. This is primarily for " +
		"debugging a long pipeline where something is going wrong and you're not sure precisely where the " +
		"state is getting broken.")
	c.AppendUsage(command + "watch-page-review" + reset + ": Watches for issues awaiting page review " +
		"(reordering or other manual processing) which are ready to be moved for " +
		"metadata entry. No job is associated with this action, hence it must run on " +
		"its own, and should only have one copy running at a time. This is also " + warning +
		"not recommended." + reset)
	c.AppendUsage(command + "watch-scans" + reset + `: Watches for issues in the "scans" folder which are ` +
		"ready to be moved for metadata entry. No job is associated with this action, " +
		"hence it must run on its own, and should only have one copy running at a time. " +
		"This is also " + warning + "not recommended." + reset)
	c.AppendUsage(command + "force-rerun" + reset + " <job id>: Creates a new job by cloning the " +
		"given job and running the new clone. This is NOT a good idea unless you know " +
		"exactly what the job(s) you're cloning can affect. This is wonderful for " +
		"testing, but should almost never be run on a production system.")

	var conf = c.GetConf()

	// run-jobs' logging defaults to Info level logs, but "-v" can make it spit
	// out debug logs. Jobs' logs written to the database are never filtered.
	if opts.Verbose {
		logLevel = ltype.Debug
	} else {
		logLevel = ltype.Info
	}
	logger.Logger = ltype.New(logLevel, false)

	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Unable to connect to the database: %s", err)
	}

	return conf, c.Args
}

// setupValidQueueNames copies in the list of valid queues for easier validation
func setupValidQueueNames() {
	for _, jType := range models.ValidJobTypes {
		var jt = string(jType)
		validQueues[jt] = true
		validQueueList = append(validQueueList, jt)
	}
}

func main() {
	setupValidQueueNames()
	var conf, args = getOpts()
	if len(args) < 1 {
		c.UsageFail("Error: you must specify an action")
	}

	// On CTRL-C / kill, try to finish the current task before exiting
	interrupts.TrapIntTerm(quit)

	// If requested, we also have a goroutine watching the jobs table so this app
	// can exit once all jobs are completed
	if opts.ExitWhenDone {
		go func() {
			// Create a brief delay so there's time for filesystem scanning to catch
			// anything that needs to be queued up. This is hacky, but auto-shutdown
			// of the job runner isn't meant for production use anyway.
			time.Sleep(time.Second * 30)
			for {
				var list, err = models.FindUnfinishedJobs()
				if err != nil {
					logger.Errorf("Unable to scan for unfinished jobs: %s", err)
				} else if len(list) == 0 {
					logger.Infof("All jobs complete, sending word to runners that it's quitting time")
					quit()
				}
				time.Sleep(time.Second)
			}
		}()
	}

	var action string
	action, args = args[0], args[1:]
	switch action {
	case "requeue":
		requeue(args)
	case "watch":
		watch(conf, args...)
	case "watch-scans":
		watchDigitizedScans(conf)
	case "watch-page-review":
		watchPageReview(conf)
	case "run-one":
		runSingleJob(conf)
	case "watchall":
		runAllQueues(conf)
	case "force-rerun":
		forceRerun(args)
	default:
		c.UsageFail("Error: invalid action")
	}
}

func requeue(ids []string) {
	if len(ids) == 0 {
		c.UsageFail("Error: the requeue action requires at least one job id")
	}

	for _, idString := range ids {
		retryJob(idString)
	}
}

func findJob(idString string) *jobs.Job {
	var id, _ = strconv.ParseInt(idString, 10, 64)
	if id == 0 {
		logger.Errorf("Invalid job id %q", idString)
		return nil
	}

	var j = jobs.Find(id)
	if j == nil {
		logger.Errorf("No job found with id %d", id)
		return nil
	}

	return j
}

func retryJob(idString string) {
	var j = findJob(idString)
	if j == nil {
		return
	}

	var failStatus = models.JobStatusFailed
	var dj = j.DBJob()
	if dj.Status != string(failStatus) {
		logger.Errorf("Cannot requeue job id %d: status is %s (it must be %s to requeue)", dj.ID, dj.Status, failStatus)
		return
	}

	logger.Infof("Requeuing job %d", dj.ID)
	var _, err = models.RenewDeadJob(dj)
	if err != nil {
		logger.Errorf("Unable to requeue job %d: %s", dj.ID, err)
	}
}

func forceRerun(ids []string) {
	if len(ids) == 0 {
		c.UsageFail("Error: the requeue action requires a job id")
	}

	if ids[0] != "we'll do it live" {
		logger.Errorf(`For safety, you must run the "force-rerun" action with an extra hidden flag. If you're not sure how to make this happen, you shouldn't be using this tool. Sorry.`)
		os.Exit(1)
	}

	if len(ids) != 2 {
		c.UsageFail("You must specify exactly one job id after the hidden flag")
	}

	rerunJob(ids[1])
}

func rerunJob(idString string) {
	var j = findJob(idString)
	if j == nil {
		return
	}

	var dj = j.DBJob()
	logger.Infof("Rerunning job %d", dj.ID)

	// Make a shallow clone of the job, strip its ID, set it to pending so it
	// runs soon, remove references to the next job to queue, but keep
	// *everything else*. This can cause massive problems if done wrong. This
	// should never be done live.
	var temp = *dj
	var clone = &temp
	clone.ID = 0
	clone.Status = string(models.JobStatusPending)
	var err = clone.Save()
	if err != nil {
		logger.Errorf("Unable to rerun job %d: %s", dj.ID, err)
	}
}

func validateJobQueue(queue string) {
	if !validQueues[queue] {
		c.UsageFail("Invalid job queue %q", queue)
	}
}

func watch(conf *config.Config, queues ...string) {
	if len(queues) == 0 {
		c.UsageFail("Error: you must specify one or more queues to watch")
	}

	var jobTypes = make([]models.JobType, len(queues))
	for i, queue := range queues {
		validateJobQueue(queue)
		jobTypes[i] = models.JobType(queue)
	}
	watchJobTypes(conf, jobTypes...)
}

func watchJobTypes(conf *config.Config, jobTypes ...models.JobType) {
	var r = jobs.NewRunner(conf, logLevel, jobTypes...)
	addRunner(r)
	r.Watch(time.Second * 10)
}

func watchPageReview(conf *config.Config) {
	logger.Infof("Watching page review folders")

	var nextAttempt time.Time
	for !done() {
		if time.Now().After(nextAttempt) {
			scanPageReviewIssues(conf)
			nextAttempt = time.Now().Add(10 * time.Minute)
		}

		// Try not to eat all the CPU
		time.Sleep(time.Second)
	}
}

func watchDigitizedScans(conf *config.Config) {
	logger.Infof("Watching in-house digitization folders")

	var nextAttempt time.Time
	for !done() {
		if time.Now().After(nextAttempt) {
			scanScannerIssues(conf)
			nextAttempt = time.Now().Add(time.Hour)
		}

		// Try not to eat all the CPU
		time.Sleep(time.Second)
	}
}

// runSingleJob simply runs a single job from the queue and exits. The
// filesystem watchers are not invoked. This is only suitable for debugging.
func runSingleJob(conf *config.Config) {
	var r = jobs.NewRunner(conf, logLevel, models.ValidJobTypes...)
	if !r.ProcessNextPendingJob() {
		logger.Infof("No pending jobs found; exiting without work")
	}
}

// runAllQueues fires up multiple goroutines to watch all the queues in a
// fairly sane way so that important processes like moving SFTP issues can
// happen quickly, while CPU-bound processes won't fight each other.
func runAllQueues(conf *config.Config) {
	waitFor(
		func() { watchPageReview(conf) },
		func() { watchDigitizedScans(conf) },
		func() {
			// Jobs which are exclusively disk IO are in the first runner to avoid
			// too much FS stuff hapenning concurrently
			watchJobTypes(conf,
				models.JobTypeArchiveBackups,
				models.JobTypeMoveDerivatives,
				models.JobTypeSyncRecursive,
				models.JobTypeVerifyRecursive,
				models.JobTypeKillDir,
				models.JobTypeWriteBagitManifest,
			)
		},
		func() {
			// Jobs which primarily use CPU are grouped next, so we aren't trying to
			// share CPU too much
			watchJobTypes(conf,
				models.JobTypePageSplit,
				models.JobTypeMakeDerivatives,
			)
		},
		func() {
			// Fast - but not instant - jobs are here: file renaming, hard-linking,
			// running templates for very simple XML output, etc. These typically
			// take very little CPU or disk IO, but they aren't "critical" jobs that
			// need to be real-time.
			watchJobTypes(conf,
				models.JobTypeBuildMETS,
				models.JobTypeCreateBatchStructure,
				models.JobTypeMakeBatchXML,
				models.JobTypeRenameDir,
				models.JobTypeCleanFiles,
				models.JobTypeRemoveFile,
				models.JobTypeWriteActionLog,
				models.JobTypeRenumberPages,
				models.JobTypeValidateTagManifest,
				models.JobTypeMarkBatchLive,
			)
		},
		func() {
			// Extremely fast data-setting jobs get a custom runner that operates
			// every second to ensure nearly real-time updates to things like a job's
			// workflow state
			var r = jobs.NewRunner(conf, logLevel,
				models.JobTypeSetIssueWS,
				models.JobTypeSetIssueBackupLoc,
				models.JobTypeSetIssueLocation,
				models.JobTypeFinalizeBatchFlaggedIssue,
				models.JobTypeEmptyBatchFlaggedIssuesList,
				models.JobTypeIgnoreIssue,
				models.JobTypeSetIssueCurated,
				models.JobTypeSetBatchStatus,
				models.JobTypeSetBatchNeedsStagingPurge,
				models.JobTypeSetBatchLocation,
				models.JobTypeIssueAction,
				models.JobTypeBatchAction,
				models.JobTypeCancelJob,
				models.JobTypeDeleteBatch,
			)
			addRunner(r)
			r.Watch(time.Second * 1)
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
