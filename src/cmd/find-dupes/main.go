// find-dupes gathers all the issue data from the find cache, and reports on
// dupes.  Maybe at some point we'll know enough to be able to actually fix the
// dupes directly.

package main

import (
	"config"
	"fmt"
	"issuewatcher"

	"os"
	"schema"
	"sort"

	"github.com/jessevdk/go-flags"
	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/wordutils"
)

// Command-line options
var opts struct {
	ConfigFile string `short:"c" long:"config" description:"path to Black Mamba config file" required:"true"`
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

var conf *config.Config

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
	var cacheFile = issuewatcher.New(conf).CacheFile()
	if !fileutil.IsFile(cacheFile) {
		usageFail("ERROR: cache-file %#v is not a valid file", cacheFile)
	}
	var watcher = issuewatcher.New(conf)
	var err = watcher.Deserialize()
	if err != nil {
		logger.Fatalf("Unable to deserialize the cache file %#v: %s", cacheFile, err)
	}

	// Store the prioritized list of issues seen for a given issue key
	var issues = make(map[string][]*schema.Issue)
	var dupeKeys []string
	var finder = watcher.IssueFinder()
	for _, issue := range finder.Issues {
		var k = issue.Key()
		// On the first dupe, we record it in the dupeKeys list
		if len(issues[k]) == 1 {
			dupeKeys = append(dupeKeys, k)
		}
		issues[k] = append(issues[k], issue)
	}

	// Strip out dupes where every "dupe" is a live issue
	//
	// TODO: Make this a flag so the "live_dupe" check below has meaning
	var keepKeys []string
	for _, k := range dupeKeys {
		var issueList = issues[k]
		var hasNonLiveIssues bool
		for _, issue := range issueList {
			if issue.Batch == nil || issue.Batch.Location[0:4] != "http" {
				hasNonLiveIssues = true
			}
		}
		if hasNonLiveIssues {
			keepKeys = append(keepKeys, k)
		}
	}
	dupeKeys = keepKeys

	sort.Strings(dupeKeys)
	for _, k := range dupeKeys {
		fmt.Printf("%#v:\n", k)
		var issueList = issues[k]
		var foundLive bool
		for i, issue := range issueList {
			var data = make(map[string]string)
			if issue.Batch != nil && issue.Batch.Location[0:4] == "http" {
				if foundLive {
					data["livedupe"] = "true"
				}
				data["production_batch"] = issue.Batch.Location
				foundLive = true
			}
			data["location"] = issue.Location
			fmt.Printf("  issue_%02d:\n", i+1)
			for k, v := range data {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
	}
}
