package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/internal/openoni"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

type _opts struct {
	cli.BaseOptions
	Environment string `long:"environment" short:"e" description:"'staging' or 'production'" required:"true"`
}

var opts _opts

const (
	cmdLoadBatch     = "load-batch"
	cmdPurgeBatch    = "purge-batch"
	cmdStatus        = "job-status"
	cmdLogs          = "job-logs"
	cmdEnsureAwardee = "ensure-awardee"
	cmdLoadTitle     = "load-title"
)

var aliases = map[string]string{
	"load":     cmdLoadBatch,
	"bl":       cmdLoadBatch,
	"purge":    cmdPurgeBatch,
	"bp":       cmdPurgeBatch,
	"stat":     cmdStatus,
	"js":       cmdStatus,
	"logs":     cmdLogs,
	"jl":       cmdLogs,
	"load-moc": cmdEnsureAwardee,
	"lmoc":     cmdEnsureAwardee,
	"lt":       cmdLoadTitle,
}

var validCmds = []string{cmdLoadBatch, cmdPurgeBatch, cmdStatus, cmdLogs, cmdEnsureAwardee, cmdLoadTitle}

func setUsage(c *cli.CLI) {
	c.AppendUsage(`Allows testing ONI Agents as well as running common commands against staging and production`)
	var aliasmap = make(map[string][]string)
	for alias, cmd := range aliases {
		aliasmap[cmd] = append(aliasmap[cmd], alias)
	}

	var all []string
	for _, cmd := range validCmds {
		var aList = aliasmap[cmd]
		sort.Strings(aList)
		var s = strings.Join(aList, " / ")
		all = append(all, fmt.Sprintf("%s (%s)", cmd, s))
	}
	c.AppendUsage("Valid commands and aliases: " + strings.Join(all, ", "))
}

func getOpts() (rpc *openoni.RPC, command string, args []string) {
	var c = cli.New(&opts)
	setUsage(c)
	var conf = c.GetConf()

	var connection string
	switch opts.Environment {
	case "staging", "stag", "s":
		connection = conf.StagingAgentConnection
	case "production", "prod", "p":
		connection = conf.ProductionAgentConnection
	default:
		c.UsageFail("Invalid environment %q", opts.Environment)
	}

	var err error
	rpc, err = openoni.New(connection)
	if err != nil {
		log.Fatalf("Unable to initialize %s ONI Agent RPC (connection string %q)", opts.Environment, connection)
	}

	if len(c.Args) == 0 {
		c.UsageFail("You must specify a command")
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
		c.UsageFail("%q is not a valid command", command)
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
	case cmdLoadBatch:
		queueJob(rpc, rpc.LoadBatch, args[0], false)
	case cmdPurgeBatch:
		queueJob(rpc, rpc.PurgeBatch, args[0], true)

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

	case cmdLoadTitle:
		if args[0] == "" {
			log.Fatalf("Invald request: you must specify a filename")
		}
		var fname = args[0]
		var data, err = os.ReadFile(fname)
		if err != nil {
			log.Fatalf("Unable to read %q: %s", fname, err)
		}
		queueJob(rpc, rpc.LoadTitle, data, true)

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

func queueJob[P any](rpc *openoni.RPC, fn func(P) (int64, error), arg P, wait bool) {
	var id, err = fn(arg)
	if err != nil {
		log.Fatalf("Couldn't queue job: %s", err)
	}
	log.Printf("Queued job: job id %d", id)

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
