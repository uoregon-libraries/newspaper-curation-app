package devtool

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
)

// command is something ncadev or ncatest can do
type command struct {
	name     string
	args     string
	help     string
	group    string
	testOnly bool
	// passthrough commands hand their arguments to something else, so they
	// aren't checked here, and a failure's exit code is ours
	passthrough bool
	run         func(s *Stack, args []string) error
}

const (
	groupStack = "Running the stack"
	groupData  = "Data"
	groupSteps = "Test steps"
	groupTest  = "Recipes"
)

var commands = []command{
	{name: "up", args: "[service...]", group: groupStack,
		help: "Start the stack, or the given services, or apply changes to them",
		run:  func(s *Stack, args []string) error { return s.Up(args...) }},
	{name: "down", group: groupStack,
		help: "Stop the stack",
		run:  noArgs((*Stack).Down)},
	{name: "build", group: groupStack,
		help: "Build the stack's images",
		run:  noArgs((*Stack).Build)},
	{name: "compose", args: "<args...>", group: groupStack, passthrough: true,
		help: `Run compose on the stack, e.g., "compose ps" or "compose restart web workers"`,
		run:  func(s *Stack, args []string) error { return s.Compose(args...) }},
	{name: "cli", args: "<command...>", group: groupStack, passthrough: true,
		help: `Run a command in a throwaway NCA container, e.g., "cli queue-batches" or "cli bash"`,
		run:  func(s *Stack, args []string) error { return s.CLI(args...) }},
	{name: "sql", args: "[mysql args...]", group: groupStack, passthrough: true,
		help: `A MariaDB console on NCA's database, or e.g. sql -e "SELECT ..."`,
		run:  func(s *Stack, args []string) error { return s.SQL(args...) }},
	{name: "workonce", group: groupStack,
		help: "Stop the workers service, and run jobs in the foreground until there are none left",
		run:  noArgs((*Stack).WorkOnce)},

	{name: "reset", group: groupData,
		help: "Wipe the databases (NCA's, both ONIs', SFTPGo's), reseed them, and start the stack",
		run:  noArgs((*Stack).Reset)},
	{name: "prep", group: groupData,
		help: "reset, plus a fresh copy of the test issues (test/sources) in the workflow tree",
		run:  noArgs((*Stack).Prep)},
	{name: "save", args: "<name>", group: groupData,
		help: "Save the stack's state (the workflow tree and data volumes)",
		run: oneArg(func(s *Stack, name string) error {
			var err = s.Save(name)
			if err == nil {
				err = s.Up()
			}
			return err
		})},
	{name: "restore", args: "<name>", group: groupData,
		help: "Replace the stack's state with a save point (or the only one whose name starts with <name>)",
		run: oneArg(func(s *Stack, name string) error {
			var err = s.Restore(name)
			if err == nil {
				err = s.Up()
			}
			return err
		})},
	{name: "nuke", group: groupData,
		help: "Stop the stack and delete its state (save points and caches are kept)",
		run:  noArgs((*Stack).Nuke)},
	{name: "init-sql", group: groupData,
		help: "Regenerate docker/init.sql from a freshly migrated database (wipes NCA's database)",
		run:  noArgs((*Stack).InitSQL)},

	{name: "rename-page-review", group: groupSteps,
		help: "Fake a page review pass: renumber issues' PDFs awaiting page review",
		run:  harnessCmd("rename-page-review")},
	{name: "make-older", args: "[days]", group: groupSteps,
		help: "Backdate issue manifests so NCA treats issues as untouched (default 4 days)",
		run:  makeOlder},
	{name: "curate", group: groupSteps,
		help: "Enter dummy metadata for every issue awaiting metadata entry",
		run:  harnessCmd("curate")},
	{name: "review", group: groupSteps,
		help: "Approve every issue awaiting metadata review",
		run:  harnessCmd("review")},
	{name: "report", args: "<name>", group: groupSteps,
		help: "Write a report of the workflow tree and database to test/<name>-report",
		run: oneArg(func(s *Stack, name string) error {
			return s.Harness("report", "--name", name)
		})},
	{name: "harness", args: "<command...>", group: groupSteps, passthrough: true,
		help: `Run any test harness command (see "harness --help")`,
		run:  func(s *Stack, args []string) error { return s.Harness(args...) }},

	{name: "run", args: "<recipe> [name]", group: groupTest, testOnly: true,
		help: `Run a recipe, naming its save points and report after [name] (default: "git describe --tags")`,
		run:  runRecipe},
	{name: "recipes", group: groupTest, testOnly: true,
		help: "List the recipes"},
}

func noArgs(fn func(*Stack) error) func(*Stack, []string) error {
	return func(s *Stack, args []string) error {
		if len(args) > 0 {
			return errUsage
		}
		return fn(s)
	}
}

func oneArg(fn func(*Stack, string) error) func(*Stack, []string) error {
	return func(s *Stack, args []string) error {
		if len(args) != 1 {
			return errUsage
		}
		return fn(s, args[0])
	}
}

func harnessCmd(name string) func(*Stack, []string) error {
	return noArgs(func(s *Stack) error { return s.Harness(name) })
}

func makeOlder(s *Stack, args []string) error {
	var days = "4"
	switch len(args) {
	case 0:
	case 1:
		if _, err := strconv.Atoi(args[0]); err != nil {
			return errUsage
		}
		days = args[0]
	default:
		return errUsage
	}
	return s.Harness("make-older", "--days", days)
}

func runRecipe(s *Stack, args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return errUsage
	}
	var rc = findRecipe(args[0])
	if rc == nil {
		return fmt.Errorf("no recipe named %q (see \"recipes\")", args[0])
	}

	var name string
	if len(args) == 2 {
		name = args[1]
	} else {
		var err error
		name, err = defaultRunName(s.Root)
		if err != nil {
			return err
		}
		fmt.Fprintf(s.Out, "No run name given; using %q\n", name)
	}
	if name == "" || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return fmt.Errorf("invalid run name %q", name)
	}
	return rc.Execute(s, name)
}

var errUsage = errors.New("usage")

func findCommand(name string, isTest bool) *command {
	for i := range commands {
		var c = &commands[i]
		if c.name == name && (isTest || !c.testOnly) {
			return c
		}
	}
	return nil
}

// Main runs the ncadev or ncatest command line: prog names the program, and
// newStack returns the stack it works on
func Main(prog string, newStack func() (*Stack, error)) {
	var isTest = prog == "ncatest"
	var args = os.Args[1:]
	if len(args) == 0 || args[0] == "help" || args[0] == "-h" || args[0] == "--help" {
		usage(os.Stdout, prog, isTest)
		return
	}

	var c = findCommand(args[0], isTest)
	if c == nil {
		fmt.Fprintf(os.Stderr, "%s: unknown command %q\n\n", prog, args[0])
		usage(os.Stderr, prog, isTest)
		os.Exit(2)
	}
	if c.name == "recipes" {
		listRecipes(os.Stdout)
		return
	}

	var s, err = newStack()
	if err == nil {
		err = c.run(s, args[1:])
	}
	if err == nil {
		return
	}

	if errors.Is(err, errUsage) {
		fmt.Fprintf(os.Stderr, "usage: %s %s %s\n", prog, c.name, c.args)
		os.Exit(2)
	}
	var exitErr *exec.ExitError
	if c.passthrough && errors.As(err, &exitErr) {
		os.Exit(exitErr.ExitCode())
	}
	fmt.Fprintf(os.Stderr, "%s: %s\n", prog, err)
	os.Exit(1)
}

func usage(w io.Writer, prog string, isTest bool) {
	if isTest {
		fmt.Fprintf(w, "%s runs NCA's test stack, a separate copy of the dev stack configured entirely\n", prog)
		fmt.Fprintln(w, "by the repository, and the test recipes that drive it. See test/README.md.")
	} else {
		fmt.Fprintf(w, "%s runs NCA's dev stack, configured by .env and compose.override.yml.\n", prog)
		fmt.Fprintln(w, "See the Developer's Guide (hugo/content/contributing/dev-guide.md).")
	}
	fmt.Fprintf(w, "\nUsage: %s <command> [args]\n", prog)

	var tw = tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	var group string
	for _, c := range commands {
		if c.testOnly && !isTest {
			continue
		}
		if c.group != group {
			group = c.group
			fmt.Fprintf(tw, "\n%s:\n", group)
		}
		fmt.Fprintf(tw, "  %s %s\t%s\n", c.name, c.args, c.help)
	}
	_ = tw.Flush()
}

func listRecipes(w io.Writer) {
	for _, rc := range recipes {
		fmt.Fprintf(w, "%s (%d stages)\n", rc.Name, len(rc.Stages))
		fmt.Fprintf(w, "  %s\n", rc.Summary)
		for i, stage := range rc.Stages {
			fmt.Fprintf(w, "  %02d: %s\n", i, stage.Name)
		}
		fmt.Fprintln(w)
	}
}
