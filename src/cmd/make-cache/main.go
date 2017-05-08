// This app uses legacyfinder to store the locations and metadata of all issues,
// batches, and titles on the filesystem or the live site.

package main

import (
	"config"
	"db"
	"fileutil"
	"fmt"
	"legacyfinder"
	"log"
	"os"
	"path/filepath"
	"wordutils"

	"github.com/jessevdk/go-flags"
)

// Conf stores the configuration data read from the legacy Python settings
var Conf *config.Config

// Command-line options
var opts struct {
	ConfigFile string `short:"c" long:"config" description:"path to P2C config file" required:"true"`
	Siteroot   string `long:"siteroot" description:"URL to the live host" required:"true"`
	CachePath  string `long:"cache-path" description:"Path to cache finder data" required:"true"`
}

var p *flags.Parser
var finder *legacyfinder.Finder

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
	fmt.Fprintln(os.Stderr)
	wrap("--cache-path will be used to cache batch/title/issue JSON in addition to " +
		"storing a dump of the finder data.  If you want to force a re-read of all " +
		"JSON, you should delete previous data in that path manually.  Note that a " +
		"site search can take a very long time.")
	fmt.Fprintln(os.Stderr)
	wrap("--siteroot must point to the live site, for downloading batch and " +
		"issue information so the search knows if an issue is live, and if so, " +
		"in what batch it was ingested.")
	os.Exit(1)
}

func getConf() {
	p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	p.Usage = "[OPTIONS]"
	var _, err = p.Parse()

	if err != nil {
		usageFail("Error: %s", err)
	}

	Conf, err = config.Parse(opts.ConfigFile)
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	err = db.Connect(Conf.DatabaseConnect)
	if err != nil {
		log.Fatalf("Error trying to connect to database: %s", err)
	}

	if !fileutil.IsDir(opts.CachePath) {
		usageFail("ERROR: --cache-path %#v is not a valid directory", opts.CachePath)
	}
}

func main() {
	getConf()
	var finder = legacyfinder.NewScanner(Conf, opts.Siteroot, opts.CachePath)

	var realFinder, err = finder.FindIssues()
	if err != nil {
		log.Fatalf("Error trying to find issues: %s", err)
	}

	var cacheFile = filepath.Join(opts.CachePath, "finder.cache")
	err = realFinder.Serialize(cacheFile)
	if err != nil {
		log.Fatalf("Error trying to serialize: %s", err)
	}
	testIntegrity(realFinder, cacheFile)
}
