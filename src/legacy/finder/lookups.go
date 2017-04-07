package main

import (
	"schema"
)

// issueMap links a textual issue key to the Issue object
type issueMap map[string][]*schema.Issue

// issueLocMap links a textual issue key to all known issue locations
type issueLocMap map[string][]string

// titleLookup lets us find titles by LCCN
var titleLookup = make(map[string]*schema.Title)

// issueLookup lets us find issues by key
var issueLookup = make(issueMap)

// issueLookupNoEdition is a lookup containing all issues for a given partial
// key, where the partial key contains everything except an Issue edition
var issueLookupNoEdition = make(issueMap)

// issueLookupNoDay looks up issues without day number or edition
var issueLookupNoDay = make(issueMap)

// issueLookupNoMonth looks up issues without month, day number, or edition
var issueLookupNoMonth = make(issueMap)

// issueLookupNoYear looks up issues without any date information
var issueLookupNoYear = make(issueMap)

// filesystemIssueLocations lets us find an issue's raw location(s)
var filesystemIssueLocations = make(issueLocMap)

// webIssueLocations tells us where an issue is located when found on the site
var webIssueLocations = make(issueLocMap)

// findOrCreateTitle looks up the given lccn to return the title, or else
// instantiates a new Title, caches it, and returns it
func findOrCreateTitle(lccn string) *schema.Title {
	var t = titleLookup[lccn]
	if t == nil {
		t = &schema.Title{LCCN: lccn}
		titleLookup[lccn] = t
	}
	return t
}

// cacheWebIssue stores the URL in the web lookup and caches the issue's key
// via cacheIssueLookup
func cacheWebIssue(i *schema.Issue, url string) {
	var k = i.Key()
	var list = webIssueLocations[k]
	list = append(list, url)
	webIssueLocations[k] = list
	cacheIssueLookup(i)
}

// cacheFilesystemIssue stores the path in the filesystem lookup and caches the
// issue's key via cacheIssueLookup
func cacheFilesystemIssue(i *schema.Issue, path string) {
	var k = i.Key()
	var list = filesystemIssueLocations[k]
	list = append(list, path)
	filesystemIssueLocations[k] = list
	cacheIssueLookup(i)
}

// cacheIssueLookup shortcuts the process of getting an issue's key and storing
// issue data in the various caches
func cacheIssueLookup(i *schema.Issue) {
	var k = i.Key()
	var iList = issueLookup[k]
	iList = append(iList, i)
	issueLookup[k] = iList

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

	// No year - which also means no slash
	k = k[:len(k)-5]
	iList = issueLookupNoYear[k]
	iList = append(iList, i)
	issueLookupNoYear[k] = iList
}
