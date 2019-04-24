package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var issueDirPattern = regexp.MustCompile(`^([a-zA-Z0-9]{10})-(\d\d\d\d)(\d\d)(\d\d)(0\d)-\d+$`)

type issuedir struct {
	parent      string
	name        string
	fullpath    string
	lccn        string
	dtfolder    string
	edition     string
	bornDigital bool
}

func newIssueDir(parent string, info os.FileInfo) (*issuedir, error) {
	var n = info.Name()
	if !info.Mode().IsDir() {
		return nil, fmt.Errorf("%q is not a directory", n)
	}

	var matches = issueDirPattern.FindStringSubmatch(n)
	if matches == nil || len(matches) < 6 {
		return nil, fmt.Errorf("name %q: doesn't match regex", n)
	}

	var idir = &issuedir{
		parent:   parent,
		name:     n,
		fullpath: filepath.Join(parent, n),
		lccn:     matches[1],
		dtfolder: fmt.Sprintf("%s-%s-%s", matches[2], matches[3], matches[4]),
		edition:  matches[5],
	}

	if idir.edition != "01" {
		idir.dtfolder += "_" + idir.edition
	}

	if _, err := time.Parse("2006-01-02", idir.dtfolder); err != nil {
		return nil, fmt.Errorf("name %q: date portion is not a valid date: %s", n, err)
	}

	var masterPath = filepath.Join(idir.fullpath, "master")
	var i, err = os.Stat(masterPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("unable to check for existence of %q: %s", masterPath, err)
	}
	if i != nil && i.Mode().IsDir() {
		idir.bornDigital = true
	}

	return idir, nil
}

type cmd struct {
	command *exec.Cmd
}
type cmdList struct {
	commands []*cmd
	live     bool
}

func command(bin string, args ...string) *cmd {
	return &cmd{exec.Command(bin, args...)}
}

func (c *cmd) String() string {
	return c.command.Path + " " + strings.Join(c.command.Args[1:], " ")
}

func (list *cmdList) append(bin string, args ...string) {
	list.commands = append(list.commands, command(bin, args...))
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <problem dir>\n", os.Args[0])
		os.Exit(1)
	}

	var probDir = os.Args[1]
	var live bool
	if len(os.Args) == 3 && os.Args[2] == "-l" {
		live = true
	}

	var infos, err = ioutil.ReadDir(probDir)
	if err != nil {
		log.Fatalf("Couldn't read %q: %s", probDir, err)
	}

	for _, info := range infos {
		log.Printf("Examining %q:", info.Name())
		var i, err = newIssueDir(probDir, info)
		if err != nil {
			log.Printf("  SKIPPING: Invalid problem directory entry: %s", err)
			continue
		}
		log.Printf("  LCCN:          %s", i.lccn)
		log.Printf("  Date folder:   %s", i.dtfolder)
		log.Printf("  Edition:       %s", i.edition)
		log.Printf("  Born Digital:  %#v", i.bornDigital)

		err = process(i, live)
		if err != nil {
			log.Fatalf("  *** Unable to process directory %q: %s", i.fullpath, err)
		}
	}
}

func (i *issuedir) listFiles(glob string) []string {
	var files, err = filepath.Glob(filepath.Join(i.fullpath, glob))
	if err != nil {
		return nil
	}

	return files
}

func process(idir *issuedir, live bool) error {
	var list = &cmdList{live: live}
	for _, file := range idir.listFiles("*.jp2") {
		list.append("rm", file)
	}
	for _, file := range idir.listFiles("*.xml") {
		list.append("rm", file)
	}

	if idir.bornDigital {
		for _, file := range idir.listFiles("*.pdf") {
			list.append("rm", file)
		}
		for _, file := range idir.listFiles("master/*.pdf") {
			list.append("mv", file, filepath.Join(idir.fullpath, filepath.Base(file)))
		}
	}
	list.append("mkdir", "-p", filepath.Join(idir.parent, idir.lccn))
	list.append("mv", idir.fullpath, filepath.Join(idir.parent, idir.lccn, idir.dtfolder))
	return list.execAll()
}

func (list *cmdList) execAll() error {
	for _, c := range list.commands {
		log.Printf("Executing %q", c)
		var err = c.exec(list.live)
		if err != nil {
			return fmt.Errorf("error executing %q: %s", c, err)
		}
	}

	return nil
}

func (c *cmd) exec(live bool) error {
	if !live {
		log.Printf("  (dry run; command not executed)")
		return nil
	}

	var output, err = c.command.CombinedOutput()
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			log.Println("  ", line)
		}
	}
	return err
}
