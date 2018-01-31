// This app reads the finder cache to report all known errors

package main

import (
	"cli"
	"issuewatcher"

	"github.com/uoregon-libraries/gopkg/logger"
)

func main() {
	var conf = cli.Simple().GetConf()
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
