package main

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/issuequeue"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

type batchQueue struct {
	currentMOC string
	mocList    []string
	mocQueue   map[string]*issuequeue.Queue
	minPages   int
	maxPages   int
}

func newBatchQueue(minPages, maxPages int) *batchQueue {
	return &batchQueue{minPages: minPages, maxPages: maxPages, mocQueue: make(map[string]*issuequeue.Queue)}
}

// FindReadyIssues looks at all issues in the database which are able to be
// batched and adds them to internal queues per MARC Org Code.  Some basic
// metadata validation takes place here as well. If redo is true, only issues
// in the "ready for rebatching" state will be considered; otherwise only
// issues in the "ready for batching" state are examined.
//
// TODO: Ensure files haven't changed (add sha checksums when issues first move
// to the metadata entry phase; store file-level info in the database so we
// have an easy checksum that's 100% separate from the filesystem)
func (q *batchQueue) FindReadyIssues(redo bool) {
	var ws schema.WorkflowStep
	if redo {
		ws = schema.WSReadyForRebatching
	} else {
		ws = schema.WSReadyForBatching
	}
	var issues, err = models.Issues().InWorkflowStep(ws).BatchID(0).Fetch()
	if err != nil {
		logger.Fatalf("Error trying to find issues: %s", err)
	}

	for _, dbIssue := range issues {
		var i, err = wrapIssue(dbIssue)
		if err != nil {
			logger.Errorf("Issue %d (%s) is invalid: %s", dbIssue.ID, dbIssue.Key(), err)
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
func (q *batchQueue) NextBatch() (*models.Batch, bool) {
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
		// This happens when the maximum batch size is too small for *any* of the
		// remaining issues in the queue
		if smallQ.pages == 0 {
			for _, i := range currentQ.list {
				logger.Debugf("Issue %q has %d pages", i.Key(), i.pages)
			}
			logger.Warnf("Cannot create a batch for %q: too many pages in all remaining issues.", q.currentMOC)
			return nil, false
		}
		if !smallQ.longWait {
			logger.Infof("Not creating a batch for %q: too few pages (%d)", q.currentMOC, smallQ.pages)
			return nil, true
		}
		logger.Infof("Small batch being pushed due to age of longest-waiting issue")
	}

	var dbIssues = make([]*models.Issue, len(smallQ.list))
	for i, issue := range smallQ.list {
		dbIssues[i] = issue.Issue
	}

	var batch, err = models.CreateBatch(conf.Webroot, q.currentMOC, dbIssues)
	if err != nil {
		logger.Fatalf("Unable to create a new batch: %s", err)
	}

	return batch, true
}
