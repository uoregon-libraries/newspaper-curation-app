package main

import (
	"log"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/openoni"
)

func getOpts() *config.Config {
	var opts cli.BaseOptions
	var c = cli.New(&opts)
	c.AppendUsage(`Tests your connections to the staging and production ONI agents`)
	var conf = c.GetConf()

	return conf
}

func main() {
	var prod, stag *openoni.RPC
	var err error

	var conf = getOpts()
	stag, err = openoni.New(conf.StagingAgentConnection)
	if err != nil {
		log.Fatalf("Can't get staging RPC: %s", err)
	}

	prod, err = openoni.New(conf.ProductionAgentConnection)
	if err != nil {
		log.Fatalf("Can't get production RPC: %s", err)
	}

	var version string
	version, err = stag.GetVersion()
	if err != nil {
		log.Fatalf("Error requesting staging version: %s", err)
	}
	log.Printf("Staging version: %q", version)

	version, err = prod.GetVersion()
	if err != nil {
		log.Fatalf("Error requesting prod version: %s", err)
	}
	log.Printf("Prod version: %q", version)

	var id int64
	id, err = stag.LoadBatch("fakey")
	if err != nil {
		log.Fatalf("Couldn't request fake batch load on staging: %s", err)
	}
	log.Printf("Got job for fake batch load on staging: job id %d", id)

	for {
		time.Sleep(time.Second)
		var jobStatus, err = stag.GetJobStatus(id)
		if err != nil {
			log.Fatalf("Couldn't request job status for fake load on staging: %s", err)
		}
		log.Printf("Got status for job %d: %s", id, jobStatus)
		if jobStatus == openoni.JobStatusSuccessful {
			break
		}
	}
}
