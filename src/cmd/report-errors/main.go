// This app reads the finder cache to report all known errors

package main

import (
	"cli"
	"issuewatcher"

	"github.com/uoregon-libraries/gopkg/logger"
)

func main() {
	var conf = cli.Simple().GetConf()
	var scanner = issuewatcher.NewScanner(conf)
	var err = scanner.Deserialize()
	if err != nil {
		logger.Fatalf("Unable to deserialize the scanner: %s", err)
	}

	// Report all errors
	for _, e := range scanner.Finder.Errors.Errors {
		logger.Errorf(e.Message())
	}
}
