// find-dupes gathers all the issue data from the find cache, and reports on
// dupes.  Maybe at some point we'll know enough to be able to actually fix the
// dupes directly.

package main

import (
	"fileutil"
	"fmt"
	"issuefinder"
	"log"
	"os"
	"schema"
	"sort"
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

	// We have to look at searchers individually so we can skip backups (duped
	// backups aren't great news, but they won't cause major problems) and try to
	// figure out a priority.  e.g., issues on the live site are almost
	// definitely canonical, and SFTP uploads are the first thing to drop if
	// there are valid dupes elsewhere.

	// Searcher order, where earlier searchers are more likely to be canonical
	var searchers = []*issuefinder.Searcher{
		finder.Searchers[issuefinder.Website],
		finder.Searchers[issuefinder.BatchedOnDisk],
		finder.Searchers[issuefinder.ReadyForBatching],
		finder.Searchers[issuefinder.AwaitingMetadataReview],
		finder.Searchers[issuefinder.AwaitingPageReview],
		finder.Searchers[issuefinder.PDFsAwaitingDerivatives],
		finder.Searchers[issuefinder.ScansAwaitingDerivatives],
		finder.Searchers[issuefinder.SFTPUpload],
	}

	// Store the prioritized list of issues seen for a given issue key
	var issues = make(map[string][]*schema.Issue)
	var dupeKeys []string
	for _, searcher := range searchers {
		for _, issue := range searcher.Issues {
			var k = issue.Key()
			// On the first dupe, we record it in the dupeKeys list
			if len(issues[k]) == 1 {
				// If this is the web searcher, we don't store dupes; there are live
				// issues which live in multiple batches.  This seems to be an error,
				// but it predates the pdf-to-chronam toolset, and needs to be handled
				// in some other way.
				if searcher == finder.Searchers[issuefinder.Website] {
					continue
				}

				dupeKeys = append(dupeKeys, k)
			}
			issues[k] = append(issues[k], issue)
		}
	}

	sort.Strings(dupeKeys)
	for _, k := range dupeKeys {
		fmt.Printf("%#v:\n", k)
		var canon, others = issues[k][0], issues[k][1:]
		fmt.Printf("  canon: %s\n", canon.Location)
		if canon.Batch != nil && canon.Batch.Location[0:4] == "http" {
			fmt.Printf("  live_batch: %s\n", canon.Batch.Location)
		}
		for i, issue := range others {
			fmt.Printf("  location%0d: %#v\n", i+2, issue.Location)
		}
	}
}
