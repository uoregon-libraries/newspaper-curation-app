package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
)

func (i *Input) makeIssueMenu() (*menu, string) {
	var m = i.makeMenu()
	m.add("reject", "Rejects the issue and sends it back for metadata entry", i.rejectIssueHandler)
	m.add("error", "Marks the issue as having an error which requires manual "+
		"intervention, pulling it off the batch and flagging it for manual intervention", i.errorIssueHandler)
	m.add("redo-derivatives", "Removes all derivative files for an issue and queues up jobs to recreate them", i.redoDerivativesHandler)
	m.add("info", "Displays some detailed information about the issue", i.issueInfoHandler)
	m.add("reload", "Reloads the issue data from the database, for seeing its latest status/information if things are changing (e.g., after running the redo-derivatives operation)", i.issueReloadHandler)
	m.add("quit", "Return to the batch menu", i.issueQuitHandler)

	return m, fmt.Sprintf("issue %q (batch %q).  Enter a command:", i.issue.db.Key(), i.batch.db.Name)
}

func (i *Input) loadIssueHandler(args []string) {
	if len(args) != 1 {
		i.printerrln(`"loadissue" must have exactly one argument: an issue id`)
		return
	}

	var id, err = strconv.Atoi(args[0])
	if err != nil {
		i.printerrln(fmt.Sprintf("%q is not a valid database id", args[0]))
		return
	}

	i.issue, err = FindIssue(id)
	if err != nil {
		i.printerrln(fmt.Sprintf("unable to load issue %d: %s", id, err.Error()))
		return
	}
	if i.issue.db.BatchID != i.batch.db.ID {
		i.printerrln(fmt.Sprintf("issue %d doesn't belong to this batch", id))
		return
	}
	i.menuFn = i.makeIssueMenu
}

func (i *Input) issueQuitHandler([]string) {
	i.issue = nil
	i.menuFn = i.makeBatchMenu
	i.println("Unloaded issue")
}

func (i *Input) rejectIssueHandler(args []string) {
	var msg = strings.Join(args, " ")
	i.println(fmt.Sprintf("%q will be removed from the batch and put back "+
		"into the metadata entry queue with a rejection message of %q.", i.issue.db.Key(), msg))
	if !i.confirmYN() {
		return
	}

	var err = i.issue.invalidateFromBatch(iTypeReject, msg)
	if err != nil {
		i.printerrln("unable to reject issue: " + err.Error())
		return
	}

	if !i.reloadBatch() {
		return
	}

	i.issue = nil
	i.menuFn = i.makeBatchMenu
	i.println("Issue has been rejected and put back on the metadata entry person's desk")
}

func (i *Input) errorIssueHandler(args []string) {
	var msg = strings.Join(args, " ")
	i.println(fmt.Sprintf("%q will be removed from the batch and moved to the \"Unfixable "+
		"Errors\" workflow tab, with an error message of %q.", i.issue.db.Key(), msg))
	if !i.confirmYN() {
		return
	}

	var err = i.issue.invalidateFromBatch(iTypeError, msg)
	if err != nil {
		i.printerrln("unable to remove issue: " + err.Error())
		return
	}

	if !i.reloadBatch() {
		return
	}

	i.issue = nil
	i.menuFn = i.makeBatchMenu
	i.println("Issue has been removed and flagged as needing manual fixes")
}

func (i *Input) redoDerivativesHandler([]string) {
	i.println(fmt.Sprintf("%q will have its derivatives regenerated.  This is an EXPERIMENTAL feature for now.  Don't do this unless you're capable of manual database cleanup!", i.issue.db.Key()))
	if !i.confirmYN() {
		return
	}

	var err = jobs.QueueForceDerivatives(i.issue.db)
	if err != nil {
		i.printerrln("unable to queue derivatives jobs: " + err.Error())
		return
	}

	i.println("Issue will have derivatives forcibly recreated.  You should NOT recreate the batch until you " + ansiIntense + "manually" + ansiReset + " verify this operation's successful completion in the database!")
}

func (i *Input) issueInfoHandler([]string) {
	var dbi = i.issue.db
	i.printDataList(
		datum{"Key", dbi.Key()},
		datum{"Title", dbi.Title.Name},
		datum{"Page Labels", dbi.PageLabelsCSV},
		datum{"Date", dbi.Date},
		datum{"Date as labeled", dbi.DateAsLabeled},
		datum{"Volume Label", dbi.Volume},
		datum{"Issue Label", dbi.Issue},
		datum{"Edition Label", dbi.EditionLabel},
		datum{"Location", dbi.Location},
		datum{"Workflow Step", dbi.WorkflowStepString},
	)
}

func (i *Input) reloadIssue() bool {
	var err error
	i.issue, err = FindIssue(i.issue.db.ID)
	if err != nil {
		i.printerrln(fmt.Sprintf("unable to reload issue %d: %s", i.issue.db.ID, err.Error()))
		i.issue = nil
		i.menuFn = i.makeBatchMenu
		return false
	}

	return true
}

func (i *Input) issueReloadHandler([]string) {
	i.reloadIssue()
}
