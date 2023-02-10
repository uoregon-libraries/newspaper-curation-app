package schema

import (
	"fmt"
	"sync"
)

// IssueMap links a textual issue key to one or more Issue objects
type IssueMap map[string]IssueList

// Lookup aggregates issue lists to create very granularly searchable data
type Lookup struct {
	sync.RWMutex

	// Issue lets us find issues by key; we should usually have only one
	// issue per key, but the live site could have something that's still sitting
	// in the "ready for ingest" area, or the page backup area.
	Issue IssueMap

	// issueNoEdition is a lookup containing all issues for a given partial
	// key, where the partial key contains everything except an Issue edition
	IssueNoEdition IssueMap

	// issueNoDay looks up issues without day number or edition
	IssueNoDay IssueMap

	// issueNoMonth looks up issues without month, day number, or edition
	IssueNoMonth IssueMap

	// issueNoYear looks up issues without any date information
	IssueNoYear IssueMap
}

// NewLookup sets up an issue key lookup for use
func NewLookup() *Lookup {
	return &Lookup{
		Issue:          make(IssueMap),
		IssueNoEdition: make(IssueMap),
		IssueNoDay:     make(IssueMap),
		IssueNoMonth:   make(IssueMap),
		IssueNoYear:    make(IssueMap),
	}
}

// Populate stores the given list of issues in the various maps
func (l *Lookup) Populate(issues IssueList) error {
	l.Lock()
	defer l.Unlock()

	var err error
	for _, issue := range issues {
		err = l.cacheIssueLookup(issue)
		if err != nil {
			return err
		}
	}

	return nil
}

// cacheIssueLookup shortcuts the process of getting an issue's key and storing
// issue data in the various caches
func (l *Lookup) cacheIssueLookup(i *Issue) error {
	var k = i.Key()

	// This shouldn't be able to happen, but a panic here blows up the whole app,
	// so let's handle it anyway
	if i.Title == nil {
		return fmt.Errorf("cannot cache issue %q (dbid %d): no title", i.Key(), i.DatabaseID)
	}

	// Normal lookup by full key
	l.Issue[k] = append(l.Issue[k], i)

	// No edition
	k = k[:len(k)-2]
	l.IssueNoEdition[k] = append(l.IssueNoEdition[k], i)

	// No day number
	k = k[:len(k)-2]
	l.IssueNoDay[k] = append(l.IssueNoDay[k], i)

	// No month
	k = k[:len(k)-2]
	l.IssueNoMonth[k] = append(l.IssueNoMonth[k], i)

	// No year - which also means no slash
	k = k[:len(k)-5]
	l.IssueNoYear[k] = append(l.IssueNoYear[k], i)

	return nil
}

// getLookup returns the appropriate issue map to use when looking up
// issues using the given key
func (l *Lookup) getLookup(k *Key) IssueMap {
	if k.Year == 0 {
		return l.IssueNoYear
	}
	if k.Month == 0 {
		return l.IssueNoMonth
	}
	if k.Day == 0 {
		return l.IssueNoDay
	}
	if k.Ed == 0 {
		return l.IssueNoEdition
	}
	return l.Issue
}

// Issues returns the list of issues which match the given search key
func (l *Lookup) Issues(k *Key) IssueList {
	l.RLock()
	defer l.RUnlock()

	var lookup = l.getLookup(k)
	var issues = lookup[k.String()]
	issues.SortByKey()
	return issues
}
