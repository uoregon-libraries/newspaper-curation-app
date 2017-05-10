// This app reads the finder cache to report all known errors

package main

import (
	"fileutil"
	"fmt"
	"issuefinder"
	"log"
	"os"
	"wordutils"

	"github.com/jessevdk/go-flags"
)

// Command-line options
var opts struct {
	CacheFile string `long:"cache-file" description:"Path to the finder cache" required:"true"`
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
	os.Exit(1)
}

func getOpts() {
	p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	var _, err = p.Parse()

	if err != nil {
		usageFail("Error: %s", err)
	}

	if !fileutil.IsFile(opts.CacheFile) {
		usageFail("ERROR: --cache-file %#v is not a valid file", opts.CacheFile)
	}
}

func main() {
	getOpts()
	var finder, err = issuefinder.Deserialize(opts.CacheFile)
	if err != nil {
		log.Fatalf("Unable to deserialize the cache file %#v: %s", opts.CacheFile, err)
	}

	finder.Errors.Index()

	// Report all errors
	for _, e := range finder.Errors.Errors {
		log.Printf("ERROR: %s", e.Message())
	}
}
