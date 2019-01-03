package main

import (
	"db"
	"fmt"
	"strconv"
)

func (i *Input) makeIssueMenu() (*menu, string) {
	var m = i.makeMenu()
	m.add("reject", "Rejects the issue and sends it back for metadata entry", i.rejectIssueHandler)
	m.add("error", "Marks the issue as having an error which requires manual "+
		"intervention, pulling it off the batch and flagging it for manual intervention", i.errorIssueHandler)
	m.add("info", "Displays some detailed information about the issue", i.issueInfoHandler)
	m.add("quit", "Return to the batch menu", i.issueQuitHandler)

	return m, fmt.Sprintf("Batch %q, issue %q.  Enter a command:", i.batch.db.Name, i.issue.db.Key())
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
	i.menuFn = i.topMenu
	i.println("Unloaded issue")
}

func (i *Input) rejectIssueHandler([]string) {
	i.printerrln("not implemented")
}

func (i *Input) errorIssueHandler([]string) {
	i.printerrln("not implemented")
}

func (i *Input) issueInfoHandler([]string) {
	i.printerrln("not implemented")
}

func (i *Input) confirmBadIssue(issue *db.Issue) {
	var msg = fmt.Sprintf("Issue %q will be removed from the batch.  Proceed (Y/N)?", issue.Key())
	var yn = i.confirm(msg, []string{"Y", "N"}, "N")
	if yn != "Y" {
		return
	}
}
