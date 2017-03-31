// chronam_json.go describes various internal structures for deserializing data
// from the live app's APIs
package main

import (
	"encoding/json"
)

// batchMetadata holds the high-level batch metadata: name and URL to query
// detailed batch information
type batchMetadata struct {
	Name string
	URL  string
}

// batchesListJSON is what we get from a batches API request.  It stores the
// list of batches' metadata and the link to the next page of results, if one
// is present.
type batchesListJSON struct {
	Batches []*batchMetadata
	Next    string
}

// issueMetadata is stored in a batch's issue list and gives us the information
// we need to query issue and title details
type issueMetadata struct {
	URL   string
	Date  string `json:"date_issued"`
	Title struct {
		URL  string
		Name string
	}
}

// batchJSON is what we get from a batch-details API request.  For our needs,
// the Issues list is all we really care about.
type batchJSON struct {
	Name   string
	Issues []*issueMetadata
}

func parseBatchJSON(encoded []byte) (*batchJSON, error) {
	var batch = &batchJSON{}
	var err = json.Unmarshal(encoded, batch)
	return batch, err
}
