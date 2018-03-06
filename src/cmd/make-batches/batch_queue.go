package main

import (
	"db"
	"log"
	"schema"
	"sort"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
)

// issue wraps a db Issue but gives us a page count as well as how old this
// issue is *relative to embargoes*
type issue struct {
	*db.Issue
	pages     int
	daysStale int
}

// issueQueue is a list of issues for a given MOC to ease batching.  It acts as
// a CS set in that you can append the same issue multiple times without having
// duplicates.
type issueQueue struct {
	list     []*issue
	seen     map[*issue]bool
	pages    int
	sorted   bool
	longWait bool
}

func newMOCIssueQueue() *issueQueue {
	var q = new(issueQueue)
	q.emptyList()
	return q
}

func (q *issueQueue) append(i *issue) {
	if q.seen[i] {
		return
	}

	q.list = append(q.list, i)
	q.pages += len(i.PageLabels)
	q.sorted = false
	q.seen[i] = true

	// Mark this queue as stale (e.g., needs batching even if we're under the
	// usual limit) if any single issue has been sitting for 30 days longer than
	// desired
	if !q.longWait {
		q.longWait = i.daysStale > 30
	}
}

func (q *issueQueue) emptyList() {
	q.seen = make(map[*issue]bool)
	q.pages = 0
	q.list = nil
	q.sorted = true
	q.longWait = false
}

// splitQueue picks the issues which will be included in the next batch, up to
// the given page limit, and puts them into a new issueQueue.  Issues are
// prioritized by those which have been waiting the longest, and then the issue
// list is iterated over multiple times to fit as many issues as possible.  The
// issues put in the returned queue are *removed* from this queue's issues list.
func (q *issueQueue) splitQueue(maxPages int) *issueQueue {
	if !q.sorted {
		sort.Slice(q.list, func(i, j int) bool {
			return q.list[i].MetadataApprovedAt.Before(q.list[j].MetadataApprovedAt)
		})
		q.sorted = true
	}

	var list = make([]*issue, len(q.list))
	copy(list, q.list)
	q.emptyList()

	var popped *issueQueue
	for passes := 3; passes > 0; passes-- {
		for _, issue := range list {
			var l = len(issue.PageLabels)
			if popped.pages+l <= maxPages {
				popped.append(issue)
			} else {
				q.append(issue)
			}
		}
	}

	return popped
}

type batchQueue struct {
	currentMOC string
	mocList    []string
	mocQueue   map[string]*issueQueue
	minPages   int
	maxPages   int
}

func newBatchQueue(minPages, maxPages int) *batchQueue {
	return &batchQueue{minPages: minPages, maxPages: maxPages, mocQueue: make(map[string]*issueQueue)}
}

// FindReadyIssues looks at all issues in the database which are able to be
// batched and adds them to internal queues per MARC Org Code.  Some basic
// metadata validation takes place here as well.
//
// TODO: Ensure files haven't changed (add sha checksums when issues first move
// to the metadata entry phase; store file-level info in the database so we
// have an easy checksum that's 100% separate from the filesystem)
func (q *batchQueue) FindReadyIssues(embargoedDays int) {
	db.LoadTitles()
	var issues, err = db.FindAvailableIssuesByWorkflowStep(schema.WSReadyForBatching)
	if err != nil {
		logger.Fatalf("Error trying to find issues: %s", err)
	}

	for _, i := range issues {
		if i.BatchID != 0 {
			continue
		}

		var key = schema.IssueKey(i.LCCN, i.Date, i.Edition)
		var issueDate, err = time.Parse("2006-01-02", i.Date)
		if err != nil {
			logger.Errorf("Issue %d (%s) has an invalid date: %s", i.ID, key, err)
			continue
		}

		var t = db.LookupTitle(i.LCCN)
		if t == nil {
			logger.Errorf("Issue %d (%s) has an LCCN with no database title", i.ID, key)
			continue
		}

		// Calculate days stale: that means the number of days that have passed
		// since this issue was allowed to be put live
		var daysStale = int(time.Since(issueDate).Hours() / 24.0)
		if t.Embargoed {
			daysStale -= embargoedDays
			if daysStale < 0 {
				logger.Infof("Skipping %s (embargoed)", key)
				continue
			}
		}

		logger.Infof("Adding %s to batch queue", key)
		var wrappedIssue = &issue{Issue: i, daysStale: daysStale}
		var mocQ, ok = q.mocQueue[wrappedIssue.MARCOrgCode]
		if !ok {
			mocQ = newMOCIssueQueue()
			q.mocQueue[wrappedIssue.MARCOrgCode] = mocQ
		}
		mocQ.append(wrappedIssue)
	}
}

func (q *batchQueue) currentQueue() (mq *issueQueue, ok bool) {
	mq = q.mocQueue[q.currentMOC]
	if len(mq.list) > 0 {
		return mq, true
	}

	delete(q.mocQueue, q.currentMOC)
	q.mocList = q.mocList[1:]
	if len(q.mocList) == 0 {
		return nil, false
	}

	q.currentMOC = q.mocList[0]
	return q.currentQueue()
}

// NextBatch returns a new Batch instance prepped with all the information
// necessary for generating a batch on disk.  Every issue put into the batch is
// removed from its queue so that each call to NextBatch returns a new batch.
// ok is false when there was nothing left to batch.
func (q *batchQueue) NextBatch() (*db.Batch, bool) {
	var currentQ, ok = q.currentQueue()
	if !ok {
		return nil, false
	}

	var smallQ = currentQ.splitQueue(q.maxPages)
	if smallQ.pages < q.minPages && !smallQ.longWait {
		return nil, false
	}

	var dbIssues = make([]*db.Issue, len(smallQ.list))
	for i, issue := range smallQ.list {
		dbIssues[i] = issue.Issue
	}

	var batch, err = db.CreateBatch(q.currentMOC, dbIssues)
	if err != nil {
		log.Fatalf("Unable to create a new batch: %s", err)
	}

	return batch, true
}
