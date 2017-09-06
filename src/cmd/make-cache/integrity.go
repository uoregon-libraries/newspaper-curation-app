package main

import (
	"fmt"
	"issuefinder"
	"logger"
)

var fails int

func integrityFail(s string) {
	fails++
	logger.Error("Integrity check failed!  %s", s)
}

func validateLen(thing string, a, b int) {
	if a == b {
		return
	}
	integrityFail(fmt.Sprintf("The %s lengths don't match; real finder had %d; cache data had %d", thing, a, b))
}

func testIntegrity(finderA *issuefinder.Finder, cacheFile string) {
	fails = 0
	logger.Info("Reading cached file to verify integrity")
	var finderB, err = issuefinder.Deserialize(cacheFile)
	if err != nil {
		integrityFail(fmt.Sprintf("Unable to deserialize the cached file: %s", err))
		return
	}

	logger.Debug("Testing deserialized finder against live finder")
	validateLen("issue", len(finderA.Issues), len(finderB.Issues))
	validateLen("title", len(finderA.Titles), len(finderB.Titles))
	validateLen("batch", len(finderA.Batches), len(finderB.Batches))
	validateLen("error", len(finderA.Errors.Errors), len(finderB.Errors.Errors))

	finderA.Issues.SortByKey()
	finderB.Issues.SortByKey()
	var issueFails int
	for i, issueA := range finderA.Issues {
		var issueB = finderB.Issues[i]

		validateLen(fmt.Sprintf("Issues[%d].Files", i), len(issueA.Files), len(issueB.Files))
		var tsvA, tsvB = issueA.TSV(), issueB.TSV()
		if tsvA != tsvB {
			issueFails++
			integrityFail(fmt.Sprintf("Issues[%d] don't match: real: %#v cache: %#v", i, tsvA, tsvB))
			if issueFails > 5 {
				break
			}
		}
	}

	finderA.Errors.Sort()
	finderB.Errors.Sort()
	var errorFails int
	for i, errorA := range finderA.Errors.Errors {
		var errorB = finderB.Errors.Errors[i]

		var msgA, msgB = errorA.Message(), errorB.Message()
		if msgA != msgB {
			errorFails++
			integrityFail(fmt.Sprintf("Errors[%d] don't match: real: %#v cache: %#v", i, msgA, msgB))
			if errorFails > 5 {
				break
			}
		}
	}

	if fails == 0 {
		logger.Info("Cache verified")
	}
}
