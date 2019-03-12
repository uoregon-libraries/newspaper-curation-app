package main

import (
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
)

// Command-line options
type _opts struct {
	cli.BaseOptions
}

var opts _opts

func getConfig() {
	var c = cli.New(&opts)
	c.AppendUsage("Allows interactive operations on batches which need to be " +
		"pushed to staging, pushed to production, requeued with problematic " +
		"issues removed, etc.")

	var conf = c.GetConf()
	var err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}
}

func main() {
	getConfig()
	var i = newInput()
	defer i.close()
	i.Listen()
}
