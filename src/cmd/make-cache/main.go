// This app uses issuewatcher to store the locations and metadata of all issues,
// batches, and titles on the filesystem or the live site.

package main

import (
	"config"
	"db"

	"fmt"
	"issuewatcher"

	"os"

	flags "github.com/jessevdk/go-flags"
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

func getConf() {
	p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	var _, err = p.Parse()
	if err != nil {
		usageFail("Error: %s", err)
	}

	conf, err = config.Parse(opts.ConfigFile)
	if err != nil {
		logger.Fatalf("Config error: %s", err)
	}

	err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}
}

func main() {
	getConf()
	var watcher = issuewatcher.New(conf)

	logger.Infof("Running scan")
	var liveFinder, err = watcher.FindIssues()
	if err != nil {
		logger.Fatalf("Error trying to find issues: %s", err)
	}

	logger.Debugf("Serializing to disk")
	err = liveFinder.Serialize(watcher.CacheFile())
	if err != nil {
		logger.Fatalf("Error trying to serialize: %s", err)
	}
	testIntegrity(liveFinder)
}
