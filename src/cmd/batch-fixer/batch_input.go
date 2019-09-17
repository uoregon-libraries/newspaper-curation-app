package main

import (
	"fmt"
	"strconv"

	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
)

func (i *Input) makeBatchMenu() (*menu, string) {
	var m = i.makeMenu()
	var st = i.batch.db.Status
	if st == db.BatchStatusQCReady || st == db.BatchStatusOnStaging {
		m.add("failqc", "Marks the batch as needing work before being put into production", i.failQCHandler)
	}
	if st == db.BatchStatusFailedQC {
		m.add("load", "Loads an issue by its id, allowing removal from the batch", i.loadIssueHandler)
		m.add("delete", "Deletes the entire batch from disk and resets associated issues in "+
			"the database to their 'ready for batching' state, for cases where a full rebatch "+
			"is easier than pulling individual issues (e.g., bad org code, dozens of bad issues, "+
			"etc.)", i.deleteBatchHandler)
	}
	if st != db.BatchStatusLive && st != db.BatchStatusLiveDone {
		m.add("list", "Lists all issues associated with this batch", i.listIssueHandler)
		m.add("search", "searches issues by various parameters.  Values are formatted as regular expressions "+
			`for the search.  e.g., "search date=19[0-6].* lccn=sn12345678 key=.*02" would find any issue `+
			"for the given lccn which was a second edition published from 1900 - 1960", i.searchIssuesHandler)
	}
	m.add("info", "Displays some detailed information about the batch", i.batchInfoHandler)
	m.add("quit", "Return to the main menu", i.batchQuitHandler)

	return m, fmt.Sprintf("Batch %q, current status: %q.  Enter a command:", i.batch.db.Name, st)
}

func (i *Input) loadBatchHandler(args []string) {
	if len(args) != 1 {
		i.printerrln(`"loadbatch" must have exactly one argument: a batch id`)
		return
	}

	var id, err = strconv.Atoi(args[0])
	if err != nil {
		i.printerrln(fmt.Sprintf("%q is not a valid database id", args[0]))
		return
	}

	i.batch, err = FindBatch(id)
	if err != nil {
		i.printerrln(fmt.Sprintf("unable to load batch %d: %s", id, err.Error()))
		return
	}

	i.menuFn = i.makeBatchMenu
}

func (i *Input) reloadBatch() bool {
	var err error
	i.batch, err = FindBatch(i.batch.db.ID)
	if err != nil {
		i.printerrln(fmt.Sprintf("unable to reload batch %d: %s", i.batch.db.ID, err))
		i.batch = nil
		i.menuFn = i.topMenu
		return false
	}

	return true
}

func (i *Input) batchQuitHandler([]string) {
	i.batch = nil
	i.menuFn = i.topMenu
	i.println("Unloaded batch")
}

func (i *Input) listIssueHandler([]string) {
	i.printIssueList(i.batch.Issues)
}

func (i *Input) printIssueList(list IssueList) {
	for _, issue := range list {
		i.println(fmt.Sprintf("  - id: %d, title: %s, key: %s", issue.db.ID, issue.db.Title.Name, issue.db.Key()))
	}
}

func (i *Input) failQCHandler([]string) {
	i.println("Removing batch...")
	var err = i.batch.Fail()
	if err != nil {
		i.printerrln("unable to remove batch: " + err.Error())
		return
	}
	i.println(`Batch removed and marked "failed_qc".  New actions are available.`)
	i.println(ansiImportant + "Right now: purge the batch from staging!" + ansiReset)
}

func (i *Input) searchIssuesHandler(args []string) {
	if len(args) == 0 {
		i.printerrln(`"searchissues" must have at least one argument, e.g., "searchissues date=200[01]0101"`)
		return
	}

	var search = new(queries)
	for _, q := range args {
		var err = search.add(q)
		if err != nil {
			i.printerrln(err.Error())
			return
		}
	}

	var matches IssueList
	for _, i := range i.batch.Issues {
		if search.match(i) {
			matches = append(matches, i)
		}
	}

	i.printIssueList(matches)
}

func (i *Input) batchInfoHandler([]string) {
	i.printDataList(
		datum{"Name", i.batch.db.FullName()},
		datum{"Location", i.batch.db.Location},
		datum{"Status", i.batch.db.Status},
		datum{"Creation", i.batch.db.CreatedAt.Format("2006-01-02 15:04:05")},
	)
}

func (i *Input) deleteBatchHandler([]string) {
	i.println(ansiImportant + "Warning" + ansiReset + ": deleting a batch is irreversible!")
	if !i.confirmYN() {
		i.println("Aborted...")
		return
	}

	i.println("Deleting batch from DB and un-associating issues")
	var err = i.batch.db.Delete()
	if err != nil {
		i.printerrln(fmt.Sprintf("Unable to update batch / issue data: %s", err))
	}
}
