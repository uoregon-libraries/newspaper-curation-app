package issuequeue

import (
	"fmt"
	"math"
	"sort"

	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// Queue is a list of issues, generally associated with a MOC, and made to
// be split into new semi-balanced queues based on a max page size
type Queue struct {
	list      []*Issue
	seen      map[string]bool
	Pages     int
	DaysStale float64
}

// New returns an issue Queue which will use the given title list to look up
// data for embargo/staleness data
func New() *Queue {
	return &Queue{seen: make(map[string]bool)}
}

// Append adds the given issue to the queue, first computing its embarge and
// stale date. If dates are bad, a title isn't found, or other metadata errors
// prevent these computations, an error is returned.
func (q *Queue) Append(issue *models.Issue) error {
	var i, err = wrapIssue(issue)
	if err != nil {
		return fmt.Errorf("wrapping issue: %w", err)
	}

	q.appendWrapped(i)
	return nil
}

// Filter returns a new Queue that includes only the elements for which filter
// returns true
func (q *Queue) Filter(filter func(*Issue) bool) *Queue {
	var newQ = New()
	for _, i := range q.list {
		if filter(i) {
			newQ.appendWrapped(i)
		}
	}

	return newQ
}

// appendWrapped adds the already-wrapped issue to a queue
func (q *Queue) appendWrapped(i *Issue) {
	if q.seen[i.Key()] {
		return
	}

	q.list = append(q.list, i)
	q.Pages += i.PageCount
	if i.DaysStale > q.DaysStale {
		q.DaysStale = i.DaysStale
	}
	q.seen[i.Key()] = true
}

// Split returns one or more new queues based off the current queue, attempting
// to split the pages as evenly as possible to achieve the least queues needed
// to satisfy the maximum page count argument. Note that maxPages is not set in
// stone: this algorithm attempts to split up queues evenly, not necessarily
// perfectly. When maxPages is exceeded, it will not be by a lot, but it should
// be seen more as a guideline than a rule.
func (q *Queue) Split(maxPages int) []*Queue {
	sort.Slice(q.list, func(i, j int) bool {
		return q.list[i].PageCount > q.list[j].PageCount
	})

	// Initialize queues
	var numQueues = int(math.Ceil(float64(q.Pages) / float64(maxPages)))
	var queues = make([]*Queue, numQueues)
	for i := range queues {
		queues[i] = New()
	}

	// Find the smallest queue to add the next issue
	for _, issue := range q.list {
		var idx = 0
		for i := 1; i < len(queues); i++ {
			if queues[i].Pages < queues[idx].Pages {
				idx = i
			}
		}
		queues[idx].appendWrapped(issue)
	}

	// Remove empty queues
	nonEmptyQueues := make([]*Queue, 0, len(queues))
	for _, q := range queues {
		if q.Pages > 0 {
			nonEmptyQueues = append(nonEmptyQueues, q)
		}
	}

	return nonEmptyQueues
}

// DBIssues returns the "unwrapped" list of issues associated with this Queue
func (q *Queue) DBIssues() []*models.Issue {
	var dbIssues = make([]*models.Issue, len(q.list))
	for i, issue := range q.list {
		dbIssues[i] = issue.Issue
	}
	return dbIssues
}

// Len returns the number of elements in the queue
func (q *Queue) Len() int {
	return len(q.list)
}
