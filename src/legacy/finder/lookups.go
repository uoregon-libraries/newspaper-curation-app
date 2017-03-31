package main

// issueMap links a textual issue key to the Issue object
type issueMap map[string][]*Issue

// issueLocMap links a textual issue key to all known issue locations
type issueLocMap map[string][]string

// titleLookup lets us find titles by LCCN
var titleLookup = make(map[string]*Title)

// issueLookup lets us find issues by key
var issueLookup = make(issueMap)

// issueLookupNoEdition is a lookup containing all issues for a given partial
// key, where the partial key contains everything except an Issue edition
var issueLookupNoEdition = make(issueMap)

// issueLookupNoDay looks up issues without day number or edition
var issueLookupNoDay = make(issueMap)

// issueLookupNoDay looks up issues without month, day number, or edition
var issueLookupNoMonth = make(issueMap)

// issueLocLookup lets us find an issue's raw location(s)
var issueLocLookup = make(issueLocMap)

// findOrCreateTitle looks up the given lccn to return the title, or else
// instantiates a new Title, caches it, and returns it
func findOrCreateTitle(lccn string) *Title {
	var t = titleLookup[lccn]
	if t == nil {
		t = &Title{LCCN: lccn}
		titleLookup[lccn] = t
	}
	return t
}

// cacheIssue shortcuts the process of getting an issue's key and storing issue
// data in the caches and issue path in the path lookup
func cacheIssue(i *Issue, location string) {
	var k = i.Key()
	var iList = issueLookup[k]
	iList = append(iList, i)
	issueLookup[k] = iList

	var ipList = issueLocLookup[k]
	ipList = append(ipList, location)
	issueLocLookup[k] = ipList

	// No edition
	k = k[:len(k)-2]
	iList = issueLookupNoEdition[k]
	iList = append(iList, i)
	issueLookupNoEdition[k] = iList

	// No day number
	k = k[:len(k)-2]
	iList = issueLookupNoDay[k]
	iList = append(iList, i)
	issueLookupNoDay[k] = iList

	// No month
	k = k[:len(k)-2]
	iList = issueLookupNoMonth[k]
	iList = append(iList, i)
	issueLookupNoMonth[k] = iList
}
