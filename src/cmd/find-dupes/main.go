// find-dupes gathers all the issue data from the find cache, and reports on
// dupes.  Maybe at some point we'll know enough to be able to actually fix the
// dupes directly.

package main

import (
	"fmt"
	"sort"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

func main() {
	var conf = cli.Simple().GetConf()
	var scanner = issuewatcher.NewScanner(conf)
	var cacheFile = scanner.CacheFile()
	if !fileutil.IsFile(cacheFile) {
		logger.Fatalf("cache-file %#v is not a valid file", cacheFile)
	}
	var err = scanner.Deserialize()
	if err != nil {
		logger.Fatalf("Unable to deserialize the cache file %#v: %s", cacheFile, err)
	}

	// Store the prioritized list of issues seen for a given issue key
	var issues = make(map[string][]*schema.Issue)
	var dupeKeys []string
	var finder = scanner.Finder
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
			if !issue.IsLive() {
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
			if issue.IsLive() {
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
