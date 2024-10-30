package main

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/openoni"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

type _opts struct {
	cli.BaseOptions
	Environment string `long:"environment" short:"e" description:"'staging' or 'production'" required:"true"`
}

var opts _opts

const (
	cmdLoad          = "load-batch"
	cmdPurge         = "purge-batch"
	cmdStatus        = "job-status"
	cmdLogs          = "job-logs"
	cmdEnsureAwardee = "ensure-awardee"
)

var aliases = map[string]string{
	"load":     cmdLoad,
	"bl":       cmdLoad,
	"purge":    cmdPurge,
	"bp":       cmdPurge,
	"stat":     cmdStatus,
	"js":       cmdStatus,
	"logs":     cmdLogs,
	"jl":       cmdLogs,
	"load-moc": cmdEnsureAwardee,
	"lmoc":     cmdEnsureAwardee,
}

var validCmds = []string{cmdLoad, cmdPurge, cmdStatus, cmdLogs}

func getOpts() (rpc *openoni.RPC, command string, args []string) {
	var c = cli.New(&opts)
	c.AppendUsage(`Allows testing ONI Agents as well as running common commands against staging and production`)
	var conf = c.GetConf()

	var connection string
	switch opts.Environment {
	case "staging", "stag", "s":
		connection = conf.StagingAgentConnection
	case "production", "prod", "p":
		connection = conf.ProductionAgentConnection
	default:
		log.Fatalf("Invalid environment %q", opts.Environment)
	}

	var err error
	rpc, err = openoni.New(connection)
	if err != nil {
		log.Fatalf("Unable to initialize %s ONI Agent RPC (connection string %q)", opts.Environment, connection)
	}

	if len(c.Args) == 0 {
		log.Fatalf("You must specify a valid command")
	}
	command, args = c.Args[0], c.Args[1:]
	if aliases[command] != "" {
		command = aliases[command]
	}
	var valid bool
	for _, cmd := range validCmds {
		if command == cmd {
			valid = true
		}
	}
	if !valid {
		log.Fatalf("Invalid command. You must choose one of: %s", strings.Join(validCmds, ", "))
	}

	var version string
	version, err = rpc.GetVersion()
	if err != nil {
		log.Fatalf("Error requesting agent version: %s", err)
	}
	log.Printf("Connected to ONI Agent on %s: version %q", opts.Environment, version)

	return rpc, command, args
}

func main() {
	var rpc, command, args = getOpts()
	if len(args) == 0 {
		args = []string{""}
	}
	switch command {
	case cmdLoad:
		doBatch(rpc, rpc.LoadBatch, args[0], false)
	case cmdPurge:
		doBatch(rpc, rpc.PurgeBatch, args[0], true)

	case cmdStatus:
		var id = getJobID(args[0])
		var js, err = rpc.GetJobStatus(id)
		if err != nil {
			log.Fatalf("Couldn't request job status: %s", err)
		}
		log.Printf("Got status for job %d: %s", id, js)

	case cmdLogs:
		var id = getJobID(args[0])
		var logs, err = rpc.GetJobLogs(id)
		if err != nil {
			log.Fatalf("Couldn't request job logs: %s", err)
		}
		for _, line := range logs {
			fmt.Println(line)
		}

	case cmdEnsureAwardee:
		if len(args) != 2 {
			log.Fatalf("Invalid request: you must specify an org code and awardee's name")
		}

		var m = &models.MOC{Code: args[0], Name: args[1]}
		var message, err = rpc.EnsureAwardee(m)
		if err != nil {
			log.Fatalf("Couldn't check/create awardee: %s", err)
		}
		fmt.Println("Success:", message)

	default:
		log.Fatalf("Command %q not handled in main", command)
	}
}

func getJobID(s string) int64 {
	var id, _ = strconv.ParseInt(s, 10, 64)
	if id < 1 {
		log.Fatalf("Invalid job id %q: must be a positive integer", s)
	}

	return id
}

type batchFunc func(name string) (jobID int64, err error)

func doBatch(rpc *openoni.RPC, fn batchFunc, batchname string, wait bool) {
	var id, err = fn(batchname)
	if err != nil {
		log.Fatalf("Couldn't request batch operation: %s", err)
	}
	log.Printf("Queued job for batch operation: job id %d", id)

	if !wait {
		log.Printf("Not waiting for long job; check status manually via the `job-status` command (job id %d)", id)
		return
	}

	for {
		var jobStatus, err = rpc.GetJobStatus(id)
		if err != nil {
			log.Fatalf("Couldn't request job status for batch process: %s", err)
		}
		log.Printf("Got status for job %d: %s", id, jobStatus)
		if jobStatus == openoni.JobStatusSuccessful {
			break
		}

		time.Sleep(time.Second)
	}
}
