// Package cli provides helpers for common Black Mamaba command-line tools' needs
package cli

import (
	"config"
	"fmt"
	"os"

	flags "github.com/jessevdk/go-flags"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/wordutils"
)

// CLI centralizes the CLI parser as well as functionality around it
type CLI struct {
	p    *flags.Parser
	opts interface{}
}

// BaseOptions represents the simplest possible list of CLI options a Black
// Mamba command can have: a config file.  Commands should extend this rather
// than having their own custom type in order to ensure consistency.
type BaseOptions struct {
	ConfigFile string `short:"c" long:"config" description:"path to Black Mamba config file" required:"true"`
}

// New returns a CLI instance for parsing flags into the given structure
func New(opts interface{}) *CLI {
	return &CLI{p: flags.NewParser(opts, flags.HelpFlag|flags.PassDoubleDash), opts: opts}
}

// Simple returns a CLI instance for parsing just a --config flag, simplifying
// the tools which don't need special-case handling
func Simple() *CLI {
	return New(&BaseOptions{})
}

// GetConf parses the command-line flags and returns the config file - it is
// assumed that the options structure can be converted to a BaseOptions value,
// otherwise this will fail
func (c *CLI) GetConf() *config.Config {
	var _, err = c.p.Parse()
	if err != nil {
		c.UsageFail("Error: %s", err)
	}

	var conf *config.Config
	conf, err = config.Parse(c.opts.(*BaseOptions).ConfigFile)
	if err != nil {
		logger.Fatalf("Config error: %s", err)
	}

	return conf
}

// Wrap is a helper to wrap a usage message at 80 characters and print a
// newline afterward
func Wrap(msg string) {
	fmt.Fprint(os.Stderr, wordutils.Wrap(msg, 80))
	fmt.Fprintln(os.Stderr)
}

// UsageFail exits the application after printing out a message and the
// parser's help
func (c *CLI) UsageFail(format string, args ...interface{}) {
	Wrap(fmt.Sprintf(format, args...))
	fmt.Fprintln(os.Stderr)
	c.p.WriteHelp(os.Stderr)
	os.Exit(1)
}
