package main

import (
	"db"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Nerdmaster/terminal"
)

const csi = "\033["
const ansiReset = csi + "0m"
const ansiIntenseYellow = csi + "33;1m"
const ansiIntenseRed = csi + "31;1m"
const ansiIntense = csi + "1m"
const errtxt = ansiIntenseRed + "Error:" + ansiReset + " "

func (i *Input) makeMenu() *menu {
	var m = new(menu)
	m.add("help", "Shows help within your current context", i.makeHelpHandler(m))
	return m
}

func (i *Input) topMenu() (*menu, string) {
	var m = i.makeMenu()
	m.add("load", "Loads a batch by id", i.loadBatchHandler)
	m.add("list", "Lists all batches that haven't gone live", i.listBatchesHandler)
	m.add("quit", "Ends the batch modification session", i.quitHandler)
	return m, "No batch or issue loaded.  Enter a command:"
}

// Input tracks the user's commands to determine what to do with the batch
// being processed, and showing the proper help menus for different states
// (batch-level vs. issue-level, for instance)
type Input struct {
	term      *terminal.Prompt
	termState *terminal.State
	inputfd   int
	done      bool
	menuFn    func() (*menu, string)
	batch     *Batch
	issue     *Issue
}

func newInput() *Input {
	var i = &Input{
		term:    terminal.NewPrompt(os.Stdin, os.Stdout, ansiIntenseYellow+"> "+ansiReset),
		inputfd: int(os.Stdin.Fd()),
	}
	i.menuFn = i.topMenu
	return i
}

// Listen loops forever, waiting for and parsing user input
func (i *Input) Listen() {
	var termState, err = terminal.MakeRaw(i.inputfd)
	if err != nil {
		panic(err)
	}
	defer i.close()

	i.termState = termState

	for !i.done {
		i.readline()
	}
}

// println sends the message to the terminal's writer (in case we ever change
// from stdout) and adds the \r\n, which is required when in raw terminal mode
func (i *Input) println(msg string) {
	i.term.Out.Write([]byte(msg))
	i.term.Out.Write([]byte("\r\n"))
}

func (i *Input) printerrln(msg string) {
	i.term.Out.Write([]byte(errtxt))
	i.println(msg)
}

func (i *Input) confirm(message string, valid []string, defaultValue string) string {
	for {
		i.println(message)
		var val = i.doReadLine()
		if i.done {
			return defaultValue
		}

		val = strings.ToUpper(val)
		if val == "" {
			return defaultValue
		}

		for _, validVal := range valid {
			if val == strings.ToUpper(validVal) {
				return val
			}
		}
		i.printerrln(fmt.Sprintf(`valid values are: %s`, strings.Join(valid, ", ")))
	}
}

type datum struct {
	name string
	val  interface{}
}

func (i *Input) printDataList(data ...datum) {
	// Determine term length so we can print with proper padding
	var termLen = 0
	for _, d := range data {
		if len(d.name) > termLen {
			termLen = len(d.name)
		}
	}

	for _, d := range data {
		var padding = termLen - len(d.name) + 1
		i.println(fmt.Sprintf("%s%s%s:%s%s", ansiIntense, d.name, ansiReset, strings.Repeat(" ", padding), d.val))
	}
}

func (i *Input) doReadLine() string {
	var raw, err = i.term.ReadLine()
	if err == nil {
		return raw
	}
	if err == io.EOF {
		return "quit"
	}

	i.done = true
	i.printerrln("cannot read input: " + err.Error())
	return ""
}

// readline gets a line of input from the user and dispatches it through the current menu
func (i *Input) readline() {
	// We rebuild the menu every time a command is entered since batch state can
	// affect options.  This should really be updated only when necessary, but
	// this whole mini-project should die when we get a real batch status page
	// built.  Hopefully.
	var m, prompt = i.menuFn()

	i.println("")
	i.println(prompt)
	var raw = i.doReadLine()
	if i.done {
		return
	}

	if !m.dispatch(raw) {
		i.printerrln(`invalid command: "` + raw + `"`)
		i.println(`Type "help" for a list of commands`)
	}
}

func (i *Input) makeHelpHandler(m *menu) handler {
	return func([]string) {
		i.println("Available commands:")
		for _, cmd := range m.commands {
			i.println(fmt.Sprintf("  - %s%s%s: %s", ansiIntense, cmd.command, ansiReset, cmd.help))
		}
	}
}

func (i *Input) quitHandler([]string) {
	i.done = true
}

func (i *Input) listBatchesHandler([]string) {
	var batches, err = db.InProcessBatches()
	if err != nil {
		i.println("unable to read batches: " + err.Error())
		return
	}
	for _, batch := range batches {
		i.println(fmt.Sprintf("  - id: %d, status: %s, name: %s", batch.ID, batch.Status, batch.FullName()))
	}
}

func (i *Input) close() {
	terminal.Restore(i.inputfd, i.termState)
}
