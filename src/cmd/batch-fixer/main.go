package main

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

// Command-line options
type _opts struct {
	cli.BaseOptions
}

var opts _opts
var conf *config.Config

func getConfig() {
	var c = cli.New(&opts)
	c.AppendUsage("Allows interactive operations on batches which need to be " +
		"pushed to staging, pushed to production, requeued with problematic " +
		"issues removed, etc.")

	conf = c.GetConf()
	var err = dbi.Connect(conf.DatabaseConnect)
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
