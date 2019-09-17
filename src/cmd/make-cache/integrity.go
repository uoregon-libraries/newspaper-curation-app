package main

import (
	"fmt"
	"sort"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuefinder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuewatcher"
)

var fails int

func integrityFail(s string) {
	fails++
	logger.Errorf("Integrity check failed!  %s", s)
}

func validateLen(thing string, a, b int) {
	if a == b {
		return
	}
	integrityFail(fmt.Sprintf("The %s lengths don't match; in-memory scanner had %d; cache data had %d", thing, a, b))
}

func testIntegrity(fA *issuefinder.Finder) {
	fails = 0
	logger.Infof("Reading cached file to verify integrity")
	var scannerB = issuewatcher.NewScanner(conf)
	var err = scannerB.Deserialize()
	if err != nil {
		integrityFail(fmt.Sprintf("Unable to deserialize the cached file: %s", err))
		return
	}

	logger.Debugf("Testing deserialized scanner against live scanner")
	var fB = scannerB.Finder
	validateLen("issue", len(fA.Issues), len(fB.Issues))
	validateLen("title", len(fA.Titles), len(fB.Titles))
	validateLen("batch", len(fA.Batches), len(fB.Batches))
	validateLen("error", len(fA.Errors), len(fB.Errors))

	logger.Debugf("Sorting issues for comparisons")
	fA.Issues.SortByKey()
	fB.Issues.SortByKey()
	logger.Debugf("Scanning %d issues to verify TSV output", len(fA.Issues))
	var issueFails int
	for i, issueA := range fA.Issues {
		var issueB = fB.Issues[i]

		validateLen(fmt.Sprintf("Issues[%d].Files", i), len(issueA.Files), len(issueB.Files))
		var tsvA, tsvB = issueA.TSV(), issueB.TSV()
		if tsvA != tsvB {
			issueFails++
			integrityFail(fmt.Sprintf("Issues[%d] don't match: in-memory: %#v cache: %#v", i, tsvA, tsvB))
			if issueFails > 5 {
				break
			}
		}
	}

	var aErr = make([]string, len(fA.Errors))
	for i, err := range fA.Errors {
		aErr[i] = err.Message()
	}

	var bErr = make([]string, len(fB.Errors))
	for i, err := range fB.Errors {
		bErr[i] = err.Message()
	}

	sort.Strings(aErr)
	sort.Strings(bErr)
	var errorFails int
	for i, errorA := range aErr {
		var errorB = bErr[i]
		if errorA != errorB {
			errorFails++
			integrityFail(fmt.Sprintf("Errors[%d] don't match: in-memory: %#v cache: %#v", i, errorA, errorB))
			if errorFails > 5 {
				break
			}
		}
	}

	if fails == 0 {
		logger.Infof("Cache verified")
	}
}
