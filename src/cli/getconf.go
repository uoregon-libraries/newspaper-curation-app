// Package cli provides helpers for common NCA command-line tools' needs
package cli

import (
	"fmt"
	"os"
	"reflect"

	flags "github.com/jessevdk/go-flags"
	"github.com/uoregon-libraries/gopkg/wordutils"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/version"
)

// CLI centralizes the CLI parser as well as functionality around it
type CLI struct {
	p         *flags.Parser
	opts      any
	Args      []string
	postUsage []string
}

// BaseOptions represents the simplest possible list of CLI options an NCA
// command can have: a config file.  Commands should extend this rather than
// having their own custom type in order to ensure consistency.
type BaseOptions struct {
	ConfigFile string `short:"c" long:"config" description:"path to NCA config file" required:"true"`
}

// New returns a CLI instance for parsing flags into the given structure
func New(opts any) *CLI {
	return &CLI{p: flags.NewParser(opts, flags.HelpFlag|flags.PassDoubleDash), opts: opts}
}

// Simple returns a CLI instance for parsing just a --config flag, simplifying
// the tools which don't need special-case handling
func Simple() *CLI {
	return New(&BaseOptions{})
}

// AppendUsage adds a string which will be printed when usage is displayed
func (c *CLI) AppendUsage(msg string) {
	c.postUsage = append(c.postUsage, msg)
}

// GetConf parses the command-line flags and returns the config file - it is
// assumed that the options structure includes a ConfigFile string (which is
// free if BaseOptions is an embedded type)
func (c *CLI) GetConf() *config.Config {
	c.Parse()

	// oV needs to be the option structure, not its pointer, so we can get its
	// enumerated fields and values
	var oV = reflect.ValueOf(c.opts).Elem()
	var fV = oV.FieldByName("ConfigFile")
	var empty reflect.Value
	if fV == empty {
		logger.Fatalf("Unable to locate ConfigFile in options structure!")
	}

	var configFile = fV.Interface().(string)

	var conf, err = config.Parse(configFile)
	if err != nil {
		logger.Fatalf("Config error: %s", err)
	}

	return conf
}

// Parse just runs the flags parser with some of our custom logic for handling
// errors and the help flag
func (c *CLI) Parse() {
	var err error
	c.Args, err = c.p.Parse()
	if err != nil {
		var ferr, ok = err.(*flags.Error)
		if ok && ferr.Type == flags.ErrHelp {
			c.HelpExit(0)
		}
		c.UsageFail("Error: %q", err)
	}
}

// Wrap is a helper to wrap a usage message at 80 characters and print a
// newline afterward
func Wrap(msg string) {
	fmt.Fprint(os.Stderr, wordutils.Wrap(msg, 80))
	fmt.Fprintln(os.Stderr)
}

// HelpExit exits the application after printing out the parser's help
func (c *CLI) HelpExit(code int) {
	fmt.Fprintln(os.Stderr, "NCA Version "+version.Version)
	fmt.Fprintln(os.Stderr)
	c.p.WriteHelp(os.Stderr)
	for _, msg := range c.postUsage {
		fmt.Fprintln(os.Stderr)
		Wrap(msg)
	}
	os.Exit(code)
}

// UsageFail exits the application after printing out a message and the
// parser's help
func (c *CLI) UsageFail(format string, args ...any) {
	Wrap(fmt.Sprintf(format, args...))
	fmt.Fprintln(os.Stderr)
	c.HelpExit(1)
}
