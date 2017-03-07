// This app looks all over the filesystem and the database to figure out if an
// issue exists somewhere in the process.  This is to help find issues we
// expected to see in production but haven't (in case they got "stuck" in some
// step) or where we have a dupe but aren't sure where all versions exist.

package main

import (
	"config"
	"db"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"wordutils"

	"github.com/jessevdk/go-flags"
)

var Conf *config.Config
var issueSearchKeys []*issueSearchKey

// Command-line options
var opts struct {
	ConfigFile string   `short:"c" long:"config" description:"path to P2C config file" required:"true"`
	IssueList  string   `long:"issue-list" description:"path to file containing list of newline-separated issue keys"`
	IssueKeys  []string `long:"issue-key" description:"single issue key to process, e.g., 'sn12345678/1905123101'"`
}

var p *flags.Parser

func usageFail(format string, args ...interface{}) {
	fmt.Fprint(os.Stderr, wordutils.Wrap(fmt.Sprintf(format, args...), 80))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr)
	p.WriteHelp(os.Stderr)
	fmt.Fprintln(os.Stderr)
	fmt.Fprint(os.Stderr, wordutils.Wrap("At least one of --issue-list or " +
		"--issue-key must be specified.  If both are specified, --issue-key will " +
		"be ignored.  Note that --issue-key may be specified multiple times.", 80))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr)
	fmt.Fprint(os.Stderr, wordutils.Wrap("Issue keys MUST be formatted as " +
		`LCCN/YYYY[MM][DD][EE].  The full LCCN and year are mandatory.`, 80))
	fmt.Fprintln(os.Stderr)
	os.Exit(1)
}

func getConf() {
	p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	p.Usage = "[OPTIONS]"
	var _, err = p.Parse()

	if err != nil {
		usageFail("Error: %s", err)
	}

	if len(opts.IssueKeys) == 0 && opts.IssueList == "" {
		usageFail("Error: You must specify one or more issue keys via --issue-keys or --issue-list")
	}

	Conf, err = config.Parse(opts.ConfigFile)
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	err = db.Connect(Conf.DatabaseConnect)
	if err != nil {
		log.Fatalf("Error trying to connect to database: %s", err)
	}

	// If we have an issue list, read it into opts.IssueKeys
	if opts.IssueList != "" {
		var contents, err = ioutil.ReadFile(opts.IssueList)
		if err != nil {
			usageFail("Unable to open issue list file %#v: %s", opts.IssueList, err)
		}
		opts.IssueKeys = strings.Split(string(contents), "\n")
	}

	// Verify that each issue key at least *looks* legit before burning time
	// searching stuff
	for _, ik := range opts.IssueKeys {
		if ik == "" {
			continue
		}

		var searchKey, err = parseSearchKey(ik)
		if err != nil {
			usageFail("Invalid issue search key %#v: %s", ik, err)
		}
		issueSearchKeys = append(issueSearchKeys, searchKey)
	}

	if len(issueSearchKeys) == 0 {
		usageFail("No valid issue keys were found (did you use a blank issue key?)")
	}
}

func main() {
	getConf()
	for _, ik := range issueSearchKeys {
		log.Printf("DEBUG: Searching for issue %#v", ik)
	}
}
