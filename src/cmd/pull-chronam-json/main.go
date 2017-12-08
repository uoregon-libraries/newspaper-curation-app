// This app just pulls from a chronam-compatible website (ONI if they remap the
// URLs like we did) to get information about all the live batches, titles,
// etc. in a cache compatible with other tools.

package main

import (
	"fmt"
	"issuefinder"

	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/wordutils"
)

// Command-line options
var opts struct {
	Siteroot  string `long:"siteroot" description:"URL to the live host" required:"true"`
	CachePath string `long:"cache-path" description:"Path to cache finder data" required:"true"`
}

var p *flags.Parser

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

	if !fileutil.IsDir(opts.CachePath) {
		usageFail("ERROR: --cache-path %#v is not a valid directory", opts.CachePath)
	}
}

func main() {
	getConf()
	var finder = issuefinder.New()

	logger.Infof("Running scan")
	var err = finder.FindWebBatches(opts.Siteroot, opts.CachePath)
	if err != nil {
		logger.Fatalf("Error trying to find issues: %s", err)
	}

	var cacheFile = filepath.Join(opts.CachePath, "finder.cache")
	logger.Debugf("Serializing to disk")
	err = finder.Serialize(cacheFile)
	if err != nil {
		logger.Fatalf("Error trying to serialize: %s", err)
	}
	testIntegrity(finder, cacheFile)
}
