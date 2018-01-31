// This app reads the finder cache to report all known errors

package main

import (
	"config"
	"fmt"
	"issuewatcher"

	"os"

	"github.com/jessevdk/go-flags"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/wordutils"
)

// Command-line options
var opts struct {
	ConfigFile string `short:"c" long:"config" description:"path to Black Mamba config file" required:"true"`
}

var p *flags.Parser
var conf *config.Config

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

func getOpts() {
	p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	var _, err = p.Parse()

	if err != nil {
		usageFail("Error: %s", err)
	}

	conf, err = config.Parse(opts.ConfigFile)
	if err != nil {
		logger.Fatalf("Config error: %s", err)
	}
}

func main() {
	getOpts()
	var watcher = issuewatcher.New(conf)
	var err = watcher.Deserialize()
	if err != nil {
		logger.Fatalf("Unable to deserialize the watcher: %s", err)
	}

	// Report all errors
	for _, e := range watcher.IssueFinder().Errors.Errors {
		logger.Errorf(e.Message())
	}
}
