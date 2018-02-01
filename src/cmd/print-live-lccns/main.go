// Downloads data from the live site to determine what LCCNs exist.  The cache
// path can be reused across other apps to reduce downloading.

package main

import (
	"cli"
	"fmt"
	"issuefinder"

	"github.com/uoregon-libraries/gopkg/logger"
)

func main() {
	var conf = cli.Simple().GetConf()

	var finder = issuefinder.New()
	var _, err = finder.FindWebBatches(conf.NewsWebroot, conf.IssueCachePath)
	if err != nil {
		logger.Fatalf("Error trying to cache live batches: %s", err)
	}
	for _, t := range finder.Titles {
		fmt.Printf("%s\t%s\t%s\n", t.LCCN, t.Name, t.PlaceOfPublication)
	}
}
