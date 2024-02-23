package main

import (
	"math"
	"strings"
	"testing"
)

var testQ *batchQueue

func setup(t *testing.T) *batchQueue {
	overrideLookup()

	testQ = newBatchQueue(1, 100)
	var dates = []string{
		"2001-01-01", "2001-02-01", "2001-03-01", "2001-04-01",
		"2001-05-01", "2001-06-01", "2001-07-01", "2001-08-01",
		"2001-09-01", "2001-10-01", "2001-11-01", "2001-12-01",
		"2002-01-01", "2002-02-01", "2002-03-01", "2002-04-01",
		"2002-05-01", "2002-06-01", "2002-07-01", "2002-08-01",
		"2002-09-01", "2002-10-01", "2002-11-01", "2002-12-01",
		"2003-01-01", "2003-02-01", "2003-03-01", "2003-04-01",
		"2003-05-01", "2003-06-01", "2003-07-01", "2003-08-01",
		"2003-09-01", "2003-10-01", "2003-11-01", "2003-12-01",
	}
	var pageLabels = "1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20"

	// moc1: 72 issues across 2 titles, 4 pages for each of first title, 6 pages for second
	var mocQ = newMOCIssueQueue()
	testQ.mocQueue["moc1"] = mocQ
	testQ.mocList = append(testQ.mocList, "moc1")
	for _, dt := range dates {
		var dbi = makeIssue(lccnSimple, dt)
		dbi.PageLabels = strings.Split(pageLabels, ",")[:4]
		var i = mustWrap(dbi, t)
		mocQ.append(i)

		dbi = makeIssue(lccnEmbargoed, dt)
		dbi.PageLabels = strings.Split(pageLabels, ",")[:6]
		i = mustWrap(dbi, t)
		mocQ.append(i)
	}

	// moc2: 12 issues, 20 pages each - only one title
	mocQ = newMOCIssueQueue()
	testQ.mocQueue["moc2"] = mocQ
	testQ.mocList = append(testQ.mocList, "moc2")
	for _, dt := range dates[:12] {
		var dbi = makeIssue(lccnEmbargoed, dt)
		dbi.PageLabels = strings.Split(pageLabels, ",")
		var i = mustWrap(dbi, t)
		mocQ.append(i)
	}
	return testQ
}

func assertEqual[T comparable](msg string, got, expected T, t *testing.T) {
	if got != expected {
		t.Errorf("%s: expected %v but got %v", msg, expected, got)
	}
}

func TestQueueing(t *testing.T) {
	setup(t)

	var queueSize = 360
	var minPageSplit = 94

	var iq, ok = testQ.currentQueue()
	if !ok {
		t.Error("testQ.currentQueue() returned not-ok")
	}
	assertEqual("testQ.currentQueue() pages", iq.pages, queueSize, t)

	// Split off batchable queues until the current queue is empty
	var splits int
	var pagesSplit int
	for iq.pages > 0 {
		splits++
		// Guard against infinite loops
		if splits > 50 {
			t.Errorf("Too many splits have occurred; aborting")
			t.FailNow()
		}

		var splitQ = iq.splitQueue(testQ.maxPages)
		pagesSplit += splitQ.pages
		t.Logf("Split number %d", splits)
		assertEqual("total pages post-split", iq.pages+pagesSplit, queueSize, t)
		assertEqual("current queue pages post-split", iq.pages, queueSize-pagesSplit, t)
		if iq.pages > 0 && (splitQ.pages < minPageSplit || splitQ.pages > testQ.maxPages) {
			t.Errorf("split queue has %d pages (should have %d to %d)", splitQ.pages, minPageSplit, testQ.maxPages)
		}
	}

	var minSplitCount = int(math.Ceil(float64(queueSize) / float64(testQ.maxPages)))
	var maxSplitCount = int(math.Ceil(float64(queueSize) / float64(minPageSplit)))
	if splits < minSplitCount || splits > maxSplitCount {
		t.Errorf("split %d times (should have been %d to %d)", splits, minSplitCount, maxSplitCount)
	}

	// currentQueue should now pull the second one

	queueSize = 240
	iq, ok = testQ.currentQueue()
	if !ok {
		t.Error("second testQ.currentQueue() returned not-ok")
	}
	assertEqual("second testQ.currentQueue() pages", iq.pages, queueSize, t)
}
