package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
)

func (i *Input) makeIssueMenu() (*menu, string) {
	var m = i.makeMenu()
	m.add("reject", "Rejects the issue and sends it back for metadata entry", i.rejectIssueHandler)
	m.add("error", "Marks the issue as having an error which requires manual "+
		"intervention, pulling it off the batch and flagging it for manual intervention", i.errorIssueHandler)
	m.add("info", "Displays some detailed information about the issue", i.issueInfoHandler)
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

	var issue *db.Issue
	issue, err = db.FindIssue(id)
	if err != nil {
		i.printerrln("database error: " + err.Error())
		return
	}

	if issue == nil {
		i.printerrln(fmt.Sprintf("issue %d wasn't found in the database", id))
		return
	}
	if issue.BatchID != i.batch.db.ID {
		i.printerrln(fmt.Sprintf("issue %d doesn't belong to this batch", id))
		return
	}

	i.issue = &Issue{db: issue}
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

	// Remove the METS file first - if this works, but the DB operation fails,
	// it's a lot easier to fix than if the DB operation succeeds but the METS
	// file is still around.
	var err = i.issue.RemoveMETS()
	if err != nil {
		i.printerrln("couldn't remove METS XML file: " + err.Error())
		return
	}

	// Save the issue's metadata
	i.issue.db.RejectMetadata(0, msg)
	i.issue.db.BatchID = 0
	err = i.issue.db.Save()
	if err != nil {
		i.printerrln("unable to update issue: " + err.Error())
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
	i.println(fmt.Sprintf("%q will be removed from the batch *and* the "+
		"workflow, with an error message of %q.", i.issue.db.Key(), msg))
	if !i.confirmYN() {
		return
	}

	// Remove the METS file first - if this works, but the DB operation fails,
	// it's a lot easier to fix than if the DB operation succeeds but the METS
	// file is still around.
	var err = i.issue.RemoveMETS()
	if err != nil {
		i.printerrln("couldn't remove METS XML file: " + err.Error())
		return
	}

	// Save the issue's metadata
	i.issue.db.ReportError(msg)
	i.issue.db.BatchID = 0
	err = i.issue.db.Save()
	if err != nil {
		i.printerrln("unable to update issue: " + err.Error())
	}

	if !i.reloadBatch() {
		return
	}

	i.issue = nil
	i.menuFn = i.makeBatchMenu
	i.println("Issue has been removed and flagged as needing manual fixes")
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
	)
}