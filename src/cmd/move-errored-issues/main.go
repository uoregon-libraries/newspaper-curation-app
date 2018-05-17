package main

import (
	"cli"
	"config"
	"db"
	"fmt"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
)

// Command-line options
type _opts struct {
	cli.BaseOptions
	Dest string `long:"destination" description:"location to move issues" required:"true"`
}

var opts _opts
var titles db.TitleList

func getOpts() *config.Config {
	var c = cli.New(&opts)
	c.AppendUsage("Finds all batches which were flagged as having errors, " +
		"moves them out of the workflow location to the given --destination, " +
		"and updates the database so the issues are no longer seen by NCA.")
	var conf = c.GetConf()
	var err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}

	if !fileutil.IsDir(opts.Dest) {
		c.UsageFail(fmt.Sprintf("Destination %q is invalid", opts.Dest))
	}

	return conf
}

func main() {
	getOpts()
	logger.Infof("Finding errored issues to move")
	var issues, err = db.FindIssuesWithErrors()
	if err != nil {
		logger.Fatalf("Unable to query the database for issues: %s", err)
	}

	for _, issue := range issues {
		fmt.Printf("mv %s %s/%s", issue.Location, opts.Dest, issue.HumanName)
	}
}
