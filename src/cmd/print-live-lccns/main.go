// Downloads data from the live site to determine what LCCNs exist.  The cache
// path can be reused across other apps to reduce downloading.

package main

import (
	"fileutil"
	"fmt"
	"issuefinder"
	"logger"
	"os"
	"wordutils"

	"github.com/jessevdk/go-flags"
)

// Command-line options
var opts struct {
	Siteroot  string `long:"siteroot" description:"URL to the live host" required:"true"`
	CachePath string `long:"cache-path" description:"Path to cache downloaded JSON files" required:"true"`
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
	wrap("--siteroot must point to the live site, for downloading batch and LCCN information")
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
	var err = finder.FindWebBatches(opts.Siteroot, opts.CachePath)
	if err != nil {
		logger.Fatal("Error trying to cache live batches: %s", err)
	}
	for _, t := range finder.Titles {
		fmt.Printf("%s\t%s\t%s\n", t.LCCN, t.Name, t.PlaceOfPublication)
	}
}
