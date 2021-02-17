package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/src/jobs"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

func (i *Input) makeBatchMenu() (*menu, string) {
	var m = i.makeMenu()
	var st = i.batch.db.Status
	if st == models.BatchStatusQCReady || st == models.BatchStatusOnStaging {
		m.add("failqc", "Marks the batch as needing work before being put into production", i.failQCHandler)
	}
	if st == models.BatchStatusFailedQC {
		m.add("load", "Loads an issue by its id, allowing removal from the batch", i.loadIssueHandler)
		m.add("removeissue", "Finds an issue by key and removes it with less interaction", i.removeIssue)
		m.add("delete", "Deletes the entire batch from disk and resets associated issues in "+
			"the database to their 'ready for batching' state, for cases where a full rebatch "+
			"is easier than pulling individual issues (e.g., bad org code, dozens of bad issues, "+
			"etc.)", i.deleteBatchHandler)
		m.add("redo-all-derivatives", "Creates jobs to rebuild *all* issues' derivatives", i.redoAllDerivatives)
		m.add("remove-all-and-delete", "Deletes the batch and removes all issues with a single error message.  This is a shortcut for removing each issue individually and then deleting the batch.  This is very dangerous and requires extra confirmation.", i.removeAllAndDelete)
		m.add("requeue", "Creates a job in the database to requeue this batch", i.requeueBatchHandler)
	}
	if st != models.BatchStatusLive && st != models.BatchStatusLiveDone {
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

func (i *Input) removeIssue(args []string) {
	var usage = func(errmsg string) {
		i.printerrln(errmsg)
		i.println("")
		i.println("usage: removeissue <type> <key> <reason>")
		i.println("")
		i.println(`type must be either "error" or "reject"`)
		i.println("key must be in the standard issuekey format: LCCN/YYYYMMDDEE")
		i.println("examples:")
		i.println("    removeissue reject sn99063854/1949012701 image 2 has a page label")
		i.println("    removeissue error sn99063854/1949012701 pages are missing - need reupload")
	}

	if len(args) < 3 {
		usage("Invalid invocation of removeissue")
		return
	}

	var returnToMetadata bool
	switch args[0] {
	case "reject":
		returnToMetadata = true
	case "error":
		returnToMetadata = false
	default:
		usage("Invalid type")
		return
	}

	var key = args[1]

	var search = new(queries)
	var err = search.add("key=" + key)
	if err != nil {
		i.printerrln(err.Error())
		return
	}

	var match *Issue
	for _, issue := range i.batch.Issues {
		if search.match(issue) {
			if match != nil {
				i.printerrln(fmt.Sprintf("More than one match for %q", key))
				return
			}

			match = issue
		}
	}

	if match == nil {
		i.printerrln(fmt.Sprintf("No issues found for %q", key))
		return
	}

	var reasonArgs = args[2:]
	i.issue = match
	if returnToMetadata {
		i.rejectIssueHandler(reasonArgs)
	} else {
		i.errorIssueHandler(reasonArgs)
	}
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

	// Auto-load a single-issue match *if* the batch is in "failed qc" status
	if len(matches) == 1 && i.batch.db.Status == models.BatchStatusFailedQC {
		i.println("Exactly one result found; automatically loading issue")
		i.issue = matches[0]
		i.menuFn = i.makeIssueMenu
	}
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

func (i *Input) redoAllDerivatives([]string) {
	i.println(fmt.Sprintf("Are you sure you want to regenerate "+ansiImportant+"EVERY"+ansiReset+
		" derivative for "+ansiIntenseYellow+"every single issue"+ansiReset+" in %s?", i.batch.db.Name))
	i.println("  (" + ansiIntenseYellow + "Note:" + ansiReset + " this can take a very long time)")
	if !i.confirmYN() {
		i.println("Aborted...")
		return
	}

	for _, issue := range i.batch.Issues {
		i.println("Queueing derivatives for " + issue.db.Key())
		var err = jobs.QueueForceDerivatives(issue.db)
		if err != nil {
			i.printerrln("failed: " + err.Error())
		}
		i.println("Success")
	}

	i.println("All issues' derivatives are now being forcibly recreated.  You should NOT recreate the batch until you " + ansiIntense + "manually" + ansiReset + " verify this operation's successful completion in the database!")
}

func (i *Input) requeueBatchHandler([]string) {
	i.println(fmt.Sprintf("Are you sure you want to requeue %s?", i.batch.db.Name))
	i.println(ansiIntenseYellow + "  (Note:" + ansiReset + " only requeue after you've removed all problem issues)")
	if !i.confirmYN() {
		i.println("Aborted...")
		return
	}

	var b = i.batch.db

	// Flag the batch as pending again to avoid confusion
	b.Status = models.BatchStatusPending
	var err = b.Save()
	if err != nil {
		i.printerrln("Unable to update batch status to 'pending' - operation aborted")
		return
	}

	// Finally: requeue
	err = jobs.QueueMakeBatch(b, conf.BatchOutputPath)
	if err != nil {
		i.printerrln(fmt.Sprintf("Error queueing batch regeneration: %s", err))
	}
}

func (i *Input) removeAllAndDelete(args []string) {
	var forceID = strconv.Itoa(i.batch.db.ID ^ 0xbead)
	var usage = func(errmsg string) {
		i.printerrln(errmsg)
		i.println("")
		i.println("usage: remove-all-and-delete <force id> <type> <reason>")
		i.println("")
		i.println(fmt.Sprintf(`<force id> must be the semi-encrypted "%s" because of how incredibly dangerous this command is.`, forceID))
		i.println(`<type> must be either "error" or "reject"`)
		i.println("examples:")
		i.println("    remove-all-and-delete <force id> reject double-check all page numbers")
		i.println("    remove-all-and-delete <force id> error missing pages")
	}

	if len(args) < 3 {
		usage("Invalid invocation of remove-all-and-delete")
		return
	}

	if args[0] != forceID {
		usage(fmt.Sprintf("<force id> must be %q", forceID))
		return
	}

	var returnToMetadata bool
	var t = args[1]
	switch t {
	case "reject":
		returnToMetadata = true
	case "error":
		returnToMetadata = false
	default:
		usage("Invalid type")
		return
	}

	var msg = strings.Join(args[2:], " ")
	i.println(fmt.Sprintf("What will happen to batch %q:", i.batch.db.FullName()))
	i.println(fmt.Sprintf("- All %d issues in this batch will be flagged as %sed", len(i.batch.Issues), t))
	i.println(fmt.Sprintf("- Issues' %q reason will be set to %q", t, msg))
	i.println("- This batch will be deleted")
	i.println("")
	i.println("Make absolutely certain that you want to do this.")
	i.println("")
	if !i.confirmYN() {
		return
	}

	for _, issue := range i.batch.Issues {
		var typ = iTypeReject
		if returnToMetadata {
			i.println("Returning to NCA: " + issue.db.Key())
		} else {
			i.println("Removing from NCA: " + issue.db.Key())
			typ = iTypeError
		}
		var err = issue.invalidateFromBatch(typ, msg)
		if err != nil {
			i.printerrln(fmt.Sprintf("Unable to invalidate issue %q: %s", issue.db.Key(), err))
			i.println("")
			i.println("This batch is all kinds of broken now.")
			i.println("Fix the problem and then re-run this command.  And hope and pray.")
			return
		}
	}

	i.println("Deleting batch from DB and un-associating issues")
	var err = i.batch.db.Delete()
	if err != nil {
		i.printerrln(fmt.Sprintf("Unable to delete batch: %s", err))
		i.println("")
		i.println("This batch is all kinds of broken now.")
		i.println("Fix the problem and then re-run this command.  And hope and pray.")
	}
}
