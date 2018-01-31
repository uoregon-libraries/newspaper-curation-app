// Package cli provides helpers for common Black Mamaba command-line tools' needs
package cli

import (
	"config"
	"fmt"
	"os"
	"reflect"

	flags "github.com/jessevdk/go-flags"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/wordutils"
)

// CLI centralizes the CLI parser as well as functionality around it
type CLI struct {
	p         *flags.Parser
	opts      interface{}
	postUsage []string
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

// AppendUsage adds a string which will be printed when usage is displayed
func (c *CLI) AppendUsage(msg string) {
	c.postUsage = append(c.postUsage, msg)
}

// GetConf parses the command-line flags and returns the config file - it is
// assumed that the options structure includes a ConfigFile string (which is
// free if BaseOptions is an embedded type)
func (c *CLI) GetConf() *config.Config {
	var _, err = c.p.Parse()
	if err != nil {
		c.UsageFail("Error: %s", err)
	}

	var configFile string

	// oV needs to be the option structure, not its pointer, so we can get its
	// enumerated fields and values
	var oV = reflect.ValueOf(c.opts).Elem()
	var fV = oV.FieldByName("ConfigFile")
	var empty reflect.Value
	if fV == empty {
		logger.Fatalf("Unable to locate ConfigFile in options structure!")
	}

	configFile = fV.Interface().(string)

	var conf *config.Config
	conf, err = config.Parse(configFile)
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
	for i, msg := range c.postUsage {
		if i > 0 {
			fmt.Fprintln(os.Stderr)
		}
		Wrap(msg)
	}
	os.Exit(1)
}
