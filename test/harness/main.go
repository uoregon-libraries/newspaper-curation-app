// Command harness runs the manual-test steps that need NCA's code, database,
// or workflow tree: loading test issues, faking page review, curation, and
// metadata review, seeding the database, and writing reports.
//
// It runs in an NCA dev container (the "cli" service), which has Go, NCA's
// settings, and the checkout mounted. scripts/reset and the test recipes run
// it for you, but you can run it yourself:
//
//	podman compose --profile tools run --rm cli go run ./test/harness <command> [options]
//
// Run it with --help for the list of commands.
package main

import (
	"errors"
	"fmt"
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
)

var opts cli.BaseOptions
var conf *config.Config

func main() {
	var p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	p.CommandHandler = func(cmd flags.Commander, args []string) error {
		if cmd == nil {
			return nil
		}
		var err = loadConfig()
		if err != nil {
			return err
		}
		return cmd.Execute(args)
	}

	addCommand(p, "load-issues", "Load the test issues",
		"Runs copy-sources, backdates the issues a month, then queues every valid born-digital issue for page "+
			"review. Stop NCA's job runner first.",
		&loadIssuesCmd{})
	addCommand(p, "copy-sources", "Replace NCA's workflow tree with the test issues",
		"Empties every one of NCA's workflow directories, then copies the test issues into NCA's upload directories.",
		&copySourcesCmd{})
	addCommand(p, "make-older", "Backdate issue manifests",
		"Rewrites the creation date in every issue manifest so NCA treats the issues as untouched for a while.",
		&makeOlderCmd{})
	addCommand(p, "rename-page-review", "Fake a page review pass",
		"Renames the PDFs of each issue awaiting page review to 0001.pdf, 0002.pdf, etc., and writes their manifests.",
		&renamePageReviewCmd{})
	addCommand(p, "curate", "Enter dummy metadata for issues awaiting metadata entry",
		"Enters dummy metadata, as the sysop user, for every issue awaiting metadata entry, and queues it for review.",
		&curateCmd{})
	addCommand(p, "review", "Approve issues awaiting metadata review",
		"Approves every issue awaiting metadata review, as the sysop user, and queues the jobs that finalize it.",
		&reviewCmd{})
	addCommand(p, "seed", "Load seed data",
		"Adds the test data's MARC org codes, replaces all users with one per role, and loads titles from MARC "+
			"records into NCA and both ONIs.",
		&seedCmd{})
	addCommand(p, "report", "Write a test report",
		"Writes a report of the workflow tree and database that's easy to diff against another run's report.",
		&reportCmd{})

	var _, err = p.Parse()
	if err != nil {
		var ferr *flags.Error
		if errors.As(err, &ferr) && ferr.Type == flags.ErrHelp {
			fmt.Println(err)
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func addCommand(p *flags.Parser, name, short, long string, cmd flags.Commander) {
	var _, err = p.AddCommand(name, short, long, cmd)
	if err != nil {
		panic(fmt.Sprintf("adding command %q: %s", name, err))
	}
}

func loadConfig() error {
	if opts.ConfigFile == "" {
		return errors.New(cli.ConfigRequiredMessage)
	}

	var err error
	conf, err = config.Parse(opts.ConfigFile)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	return nil
}

func connectDB() error {
	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		return fmt.Errorf("connecting to the database: %w", err)
	}
	return nil
}
