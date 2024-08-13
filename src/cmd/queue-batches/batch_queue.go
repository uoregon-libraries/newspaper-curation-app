package main

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/issuequeue"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
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

	for _, i := range issues {
		var moc = i.MARCOrgCode
		var mocQ, ok = q.mocQueue[moc]
		if !ok {
			mocQ = issuequeue.New(titles)
			q.mocQueue[moc] = mocQ
			q.mocList = append(q.mocList, moc)
		}

		var err = mocQ.Append(i)
		if err != nil {
			logger.Errorf("Cannot queue issue %d (%s): %s", i.ID, i.Key(), err)
		}
	}
}

// CreateBatches iterates over the issue queues, splits them where necessary,
// and returns batches stored in the DB and ready for processing.
//
// The queues are pre-processed before splitting and batch building in order to
// remove Issues which are embargoed
func (q *batchQueue) CreateBatches(seed string) []*models.Batch {
	// Step 1: clean up queues
	var queues []*issuequeue.Queue
	for moc, mocQueue := range q.mocQueue {
		var newQ = mocQueue.RemoveIf(func(i *issuequeue.Issue) bool {
			if i.Embargoed {
				logger.Debugf("Removing issue %q from %s queue: embargoed", i.Key(), moc)
				return true
			}
			return false
		})

		// This can happen if all issues are embargoed
		if newQ.Pages == 0 {
			continue
		}

		if newQ.Pages < q.minPages {
			if newQ.DaysStale < 30 {
				logger.Debugf("Small queue %q (%d pages): skipping", moc, newQ.Pages)
				continue
			}

			logger.Debugf("Small queue %q (%d pages): pushed due to age", moc, newQ.Pages)
		}

		queues = append(queues, newQ)
	}

	// Step 2: split queues by max page count, and turn them into batches
	var batches []*models.Batch
	for _, q2 := range queues {
		for _, next := range q2.Split(q.maxPages) {
			var dbIssues = next.DBIssues()
			var batch, err = models.CreateBatch(seed, dbIssues[0].MARCOrgCode, dbIssues)
			if err != nil {
				logger.Fatalf("Unable to create a new batch: %s", err)
			}
			batches = append(batches, batch)
		}
	}

	return batches
}
