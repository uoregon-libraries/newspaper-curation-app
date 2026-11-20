package devtool

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Recipe is a scripted test run on the test stack: a series of stages, each
// followed by a save point, then a report (see the harness's "report"
// command). Rerunning a recipe with the same name picks up after the last
// stage that has a save point, so an interrupted run, or one that's being
// repeated from some midpoint, doesn't start over.
//
// Some stages pause for the tester to check something, or break something,
// by hand.
type Recipe struct {
	Name    string
	Summary string
	Stages  []Stage
}

// Stage is one step of a recipe
type Stage struct {
	Name string
	Run  func(*recipeRun) error
}

// recipeRun is a recipe in progress
type recipeRun struct {
	s  *Stack
	in *bufio.Reader
}

// pause waits for the tester to press Enter
func (r *recipeRun) pause(msg string) error {
	fmt.Fprintf(r.s.Out, "\n%s\n\nPress Enter to continue: ", msg)
	var _, err = r.in.ReadString('\n')
	if errors.Is(err, io.EOF) {
		return errors.New("no input: recipes that pause need a terminal")
	}
	return err
}

func (r *recipeRun) workonce() error {
	return r.s.WorkOnce()
}

func (r *recipeRun) harness(args ...string) error {
	return r.s.Harness(args...)
}

// recipes are all the recipes ncatest knows. Every recipe here is a variation
// on the general test, so they share most of their stages.
var recipes = []*Recipe{
	{
		Name: "general",
		Summary: "End to end, from nothing: process uploads, fake curation and review, then build and approve " +
			"batches. A good baseline for refactors, and a starting point for new recipes.",
		Stages: []Stage{prepStage, processStage, curateStage, batchStage, goLiveStage},
	},
	{
		Name:    "invalid-title",
		Summary: "The general test, but one title's LCCN is marked invalid before batching, so its issues must be kept out of batches.",
		Stages:  []Stage{prepStage, processStage, curateStage, invalidTitleBatchStage, goLiveStage},
	},
	{
		Name: "broken-oni",
		Summary: "The general test, but you stop staging ONI's Solr while batches load, to test how NCA handles " +
			"ONI jobs failing (entwined job retries).",
		Stages: []Stage{prepStage, processStage, curateStage, brokenONIBatchStage, goLiveStage},
	},
}

var prepStage = Stage{
	Name: "Reset the stack and load the test issues",
	Run:  func(r *recipeRun) error { return r.s.prepData() },
}

var processStage = Stage{
	Name: "Process uploads, then fake page review",
	Run: func(r *recipeRun) error {
		var err = r.workonce()
		if err == nil {
			err = r.harness("rename-page-review")
		}
		if err == nil {
			err = r.harness("make-older", "--days", "4")
		}
		// The job runner rescans the filesystem now and then; restarting it
		// gets the renamed issues moving right away
		if err == nil {
			err = r.workonce()
		}
		return err
	},
}

var curateStage = Stage{
	Name: "Fake curation and metadata review",
	Run: func(r *recipeRun) error {
		var err = r.harness("curate")
		if err == nil {
			err = r.workonce()
		}
		if err == nil {
			err = r.harness("review")
		}
		if err == nil {
			err = r.workonce()
		}
		return err
	},
}

var batchStage = Stage{
	Name: "Build batches and load them into staging",
	Run:  queueBatches,
}

func queueBatches(r *recipeRun) error {
	var err = r.s.CLI("queue-batches")
	if err == nil {
		err = r.workonce()
	}
	if err == nil {
		err = r.pause(verifyStaging(r.s))
	}
	return err
}

func verifyStaging(s *Stack) string {
	return fmt.Sprintf("Check that the batches are on staging ONI (%s), then approve them in NCA (%s/nca?debuguser=sysop).",
		s.testURL("ONI_STAGING_PORT"), s.testURL("PUBLIC_PORT"))
}

var invalidTitleBatchStage = Stage{
	Name: "Invalidate a title's LCCN, then build batches and load them into staging",
	Run: func(r *recipeRun) error {
		fmt.Fprintln(r.s.Out, "Marking one title with issues ready for batching as having an invalid LCCN")
		var err = r.s.SQL("-e", `
			UPDATE titles
			SET valid_lccn = 0
			WHERE lccn IN (
				SELECT lccn FROM issues WHERE workflow_step = 'ReadyForBatching'
				ORDER BY lccn
			)
			LIMIT 1`)
		if err == nil {
			err = queueBatches(r)
		}
		return err
	},
}

var brokenONIBatchStage = Stage{
	Name: "Build batches, and break staging ONI while they load",
	Run: func(r *recipeRun) error {
		var err = r.s.CLI("queue-batches")
		if err == nil {
			err = r.pause("Stop staging ONI's Solr. In another terminal:\n\n    bin/ncatest compose stop oni-staging-solr")
		}
		if err == nil {
			err = r.pause("Next, the job runner starts, and loading batches into staging fails. Once you've seen " +
				"enough \"oni_wait_for_job\" failures to be confident in the retries, start Solr again so the jobs " +
				"can finish:\n\n    bin/ncatest compose start oni-staging-solr")
		}
		if err == nil {
			err = r.workonce()
		}
		if err == nil {
			err = r.pause(verifyStaging(r.s))
		}
		return err
	},
}

var goLiveStage = Stage{
	Name: "Load the approved batches into production",
	Run: func(r *recipeRun) error {
		var err = r.workonce()
		if err == nil {
			err = r.pause(fmt.Sprintf("Check that the batches are on production ONI (%s), and are \"live\" in NCA.",
				r.s.testURL("ONI_PROD_PORT")))
		}
		return err
	},
}

func findRecipe(name string) *Recipe {
	for _, rc := range recipes {
		if rc.Name == name {
			return rc
		}
	}
	return nil
}

// savepointName is the save point for a run's stage, e.g., "02-baseline"
func savepointName(stage int, runName string) string {
	return fmt.Sprintf("%02d-%s", stage, runName)
}

// defaultRunName names a run for the checked-out code
func defaultRunName(root string) (string, error) {
	var cmd = exec.Command("git", "describe", "--tags")
	cmd.Dir = root
	var out, err = cmd.Output()
	if err != nil {
		return "", fmt.Errorf("naming the run with \"git describe --tags\" failed (%w): give it a name", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Execute runs the recipe on s (which must be the test stack), naming its
// save points and report after runName
func (rc *Recipe) Execute(s *Stack, runName string) error {
	var reportDir = filepath.Join(s.Root, "test", runName+"-report")
	if _, err := os.Stat(reportDir); err == nil {
		return fmt.Errorf("%s already exists: remove or rename it, or name this run something else", reportDir)
	}

	// Everything compose (and so everything in a container) writes goes to
	// the terminal and a log, so the run can be checked afterward
	var logPath = filepath.Join(s.Root, "test", "var", "logs", "recipe-"+runName+".log")
	var err = os.MkdirAll(filepath.Dir(logPath), 0755)
	if err != nil {
		return err
	}
	var log *os.File
	log, err = os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer log.Close()
	s.Out = io.MultiWriter(os.Stdout, log)
	s.Err = io.MultiWriter(os.Stderr, log)
	fmt.Fprintf(s.Out, "Running recipe %q as %q (log: %s)\n", rc.Name, runName, logPath)

	var r = &recipeRun{s: s, in: bufio.NewReader(os.Stdin)}
	err = s.Build()
	if err != nil {
		return err
	}

	// Pick up after the last stage that has a save point
	var start = 0
	for i := len(rc.Stages) - 1; i >= 0; i-- {
		var sp = savepointName(i, runName)
		if s.HasSavepoint(sp) {
			fmt.Fprintf(s.Out, "\nFound save point %s; restoring it to skip ahead\n", sp)
			err = s.Restore(sp)
			if err == nil {
				err = s.upForRecipe()
			}
			if err != nil {
				return err
			}
			start = i + 1
			break
		}
	}

	for i := start; i < len(rc.Stages); i++ {
		var stage = rc.Stages[i]
		fmt.Fprintf(s.Out, "\n=== Stage %02d: %s ===\n\n", i, stage.Name)
		if i > 0 {
			err = s.CLI("wait-for-db")
			if err != nil {
				return err
			}
		}

		err = stage.Run(r)
		if err != nil {
			return fmt.Errorf("stage %02d (%s): %w", i, stage.Name, err)
		}

		err = s.Save(savepointName(i, runName))
		if err == nil {
			err = s.upForRecipe()
		}
		if err != nil {
			return err
		}
	}

	err = s.CLI("wait-for-db")
	if err == nil {
		err = s.Harness("report", "--name", runName)
	}
	if err != nil {
		return err
	}

	fmt.Fprintf(s.Out, "\nDone. The report is in %s\n", reportDir)
	fmt.Fprintln(s.Out, `The workers service is stopped; "bin/ncatest up" starts it.`)
	return nil
}

// upForRecipe starts everything but the workers service. Recipes run jobs
// with WorkOnce, at set points; a workers service starting on its own would
// race them, making runs less repeatable.
func (s *Stack) upForRecipe() error {
	return s.upExcept("workers")
}
