// This app reads the finder cache to report all known errors

package main

import (
	"fmt"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
)

func main() {
	var conf = cli.Simple().GetConf()

	var err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}

	var scanner = issuewatcher.NewScanner(conf)
	err = scanner.Deserialize()
	if err != nil {
		logger.Fatalf("Unable to deserialize the scanner: %s", err)
	}

	// Report all errors
	reportErrors("Root", scanner.Finder.Errors)
	for _, b := range scanner.Finder.Batches {
		reportErrors(fmt.Sprintf("Batch %q (%q)", b.Fullname(), b.Location), b.Errors)
	}
	for _, t := range scanner.Finder.Titles {
		reportErrors(fmt.Sprintf("Title %q (%s)", t.Name, t.LCCN), t.Errors)
		for _, i := range t.Issues {
			reportErrors(fmt.Sprintf("Issue %s", i.Location), i.Errors)
			for _, f := range i.Files {
				reportErrors(fmt.Sprintf("File %s/%s", i.Location, f.Name), f.Errors)
			}
		}
	}
}

func reportErrors(title string, list apperr.List) {
	if len(list) == 0 {
		return
	}

	fmt.Printf("- %s\n", title)
	for _, err := range list {
		fmt.Printf("  - %s\n", err.Message())
	}
}
