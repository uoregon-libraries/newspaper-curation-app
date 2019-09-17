package main

import (
	"sort"

	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

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
	q.pages += i.pages
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
// prioritized by those which have been waiting the longest, and then issues
// are added to the new queue.  Issues put in the returned queue are *removed*
// from this queue's issues list.
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

	var popped = newMOCIssueQueue()
	for _, issue := range list {
		var l = len(issue.PageLabels)
		if popped.pages+l <= maxPages {
			popped.append(issue)
		} else {
			q.append(issue)
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
func (q *batchQueue) FindReadyIssues() {
	var issues, err = db.FindIssuesReadyForBatching()
	if err != nil {
		logger.Fatalf("Error trying to find issues: %s", err)
	}

	for _, dbIssue := range issues {
		var i, err = wrapIssue(dbIssue)
		if err != nil {
			logger.Errorf("Issue %d (%s) is invalid: %s", i.ID, i.Key(), err)
			continue
		}

		if i.embargoed {
			logger.Infof("Skipping %s (embargoed)", i.Key())
			continue
		}

		logger.Infof("Adding %s to batch queue", i.Key())
		var moc = i.MARCOrgCode
		var mocQ, ok = q.mocQueue[moc]
		if !ok {
			mocQ = newMOCIssueQueue()
			q.mocQueue[moc] = mocQ
			q.mocList = append(q.mocList, moc)
		}
		mocQ.append(i)
	}
}

// nextMOC calculates which MARC Org Code should be used for the next batch and
// returns its issue queue.  Iterates through known MOCs when queues are empty
// until a queue is found or no queues are left, in which case nil is returned.
func (q *batchQueue) nextMOCQueue() *issueQueue {
	var mq = q.mocQueue[q.currentMOC]
	if mq != nil && len(mq.list) > 0 {
		return mq
	}

	if len(q.mocList) == 0 {
		return nil
	}

	q.currentMOC, q.mocList = q.mocList[0], q.mocList[1:]
	return q.nextMOCQueue()
}

func (q *batchQueue) currentQueue() (*issueQueue, bool) {
	var mq = q.nextMOCQueue()
	return mq, mq != nil
}

// NextBatch returns a new Batch instance prepped with all the information
// necessary for generating a batch on disk.  Every issue put into the batch is
// removed from its queue so that each call to NextBatch returns a new batch.
// ok is false when there was nothing left to batch.
func (q *batchQueue) NextBatch() (*db.Batch, bool) {
	for moc, mq := range q.mocQueue {
		if mq.pages > 0 {
			logger.Debugf("%q queue has %d pages left", moc, mq.pages)
		}
	}

	var currentQ, ok = q.currentQueue()
	if !ok {
		logger.Debugf("Operation complete: no issues were found in the remaining queue(s)")
		return nil, false
	}

	var smallQ = currentQ.splitQueue(q.maxPages)
	if smallQ.pages < q.minPages {
		if !smallQ.longWait {
			logger.Infof("Not creating a batch for %q: too few pages (%d)", q.currentMOC, smallQ.pages)
			return nil, true
		}
		logger.Infof("Small batch being pushed due to age of longest-waiting issue")
	}

	var dbIssues = make([]*db.Issue, len(smallQ.list))
	for i, issue := range smallQ.list {
		dbIssues[i] = issue.Issue
	}

	var batch, err = db.CreateBatch(conf.Webroot, q.currentMOC, dbIssues)
	if err != nil {
		logger.Fatalf("Unable to create a new batch: %s", err)
	}

	return batch, true
}
