package chronam

import (
	"encoding/json"
)

// BatchMetadata holds the high-level batch metadata: name and URL to query
// detailed batch information
type BatchMetadata struct {
	Name string
	URL  string
}

// BatchesListJSON is what we get from a batches API request.  It stores the
// list of batches' metadata and the link to the next page of results, if one
// is present.
type BatchesListJSON struct {
	Batches []*BatchMetadata
	Next    string
}

// IssueMetadata is stored in a batch's issue list and gives us the information
// we need to query issue and title details
type IssueMetadata struct {
	URL   string
	Date  string `json:"date_issued"`
	Title struct {
		URL  string
		Name string
	}
}

// TitleJSON is what we get from a title-details API request
type TitleJSON struct {
	LCCN  string
	Name  string
	Place string `json:"place_of_publications"`
}

// BatchJSON is what we get from a batch-details API request
type BatchJSON struct {
	Name   string
	Issues []*IssueMetadata
	LCCNs  []string
}

// ParseBatchJSON takes a pile of bytes and attempts to convert them into a
// BatchJSON structure.  If json.Unmarshal has an error, it will be returned
// along with a nil object.
func ParseBatchJSON(encoded []byte) (*BatchJSON, error) {
	var batch = &BatchJSON{}
	var err = json.Unmarshal(encoded, batch)
	return batch, err
}

// ParseBatchesListJSON takes a pile of bytes and attempts to convert them into
// a BatchesListJSON structure.  If json.Unmarshal has an error, it will be
// returned along with a nil object.
func ParseBatchesListJSON(encoded []byte) (*BatchesListJSON, error) {
	var bList = &BatchesListJSON{}
	var err = json.Unmarshal(encoded, bList)
	return bList, err
}

// ParseBatchesListJSON takes a pile of bytes and attempts to convert them into
// a TitleJSON structure.  If json.Unmarshal has an error, it will be returned
// along with a nil object.
func ParseTitleJSON(encoded []byte) (*TitleJSON, error) {
	var t = &TitleJSON{}
	var err = json.Unmarshal(encoded, t)
	return t, err
}
