package issuefinder

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"schema"
)

// Deserialize attempts to read and deserialize the given filename into a
// Finder, returning the Finder if successful, nil and an error otherwise
func Deserialize(filename string) (*Finder, error) {
	var content, err = ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %#v: %s", filename, err)
	}

	var dec = gob.NewDecoder(bytes.NewBuffer(content))
	var cf cachedFinder
	err = dec.Decode(&cf)
	if err != nil {
		return nil, fmt.Errorf("unable to deserialize %#v: %s", filename, err)
	}

	return cf.finder(), nil
}

// finder internally converts the cache-friendly data to a Finder instance
func (cf cachedFinder) finder() *Finder {
	var batchLookup = make(map[cacheID]*schema.Batch)
	var titleLookup = make(map[cacheID]*schema.Title)
	var issueLookup = make(map[cacheID]*schema.Issue)
	var f = New()

	// Build the basic schema objects with associations
	for _, cb := range cf.Batches {
		var b = &schema.Batch{
			MARCOrgCode: cb.MARCOrgCode,
			Keyword:     cb.Keyword,
			Version:     cb.Version,
			Location:    cb.Location,
		}
		batchLookup[cb.ID] = b
		f.Batches = append(f.Batches, b)
	}
	for _, ct := range cf.Titles {
		var t = &schema.Title{
			LCCN:               ct.LCCN,
			Name:               ct.Name,
			PlaceOfPublication: ct.PlaceOfPublication,
			Location:           ct.Location,
		}
		titleLookup[ct.ID] = t
		f.Titles = append(f.Titles, t)
		f.titleByLoc[t.Location] = t
	}
	for _, ci := range cf.Issues {
		var i = &schema.Issue{
			Date:     ci.Date,
			Edition:  ci.Edition,
			Location: ci.Location,
		}
		issueLookup[ci.ID] = i
		f.Issues = append(f.Issues, i)

		// Associate the title and batch
		batchLookup[ci.BatchID].AddIssue(i)
		titleLookup[ci.TitleID].AddIssue(i)
	}

	// Populate the Errors list
	for _, ce := range cf.Errors {
		var e = &Error{
			Location: ce.Location,
			Error:    fmt.Errorf(ce.Error),
		}
		if ce.BatchID != 0 {
			e.Batch = batchLookup[ce.BatchID]
		}
		if ce.TitleID != 0 {
			e.Title = titleLookup[ce.TitleID]
		}
		if ce.IssueID != 0 {
			e.Issue = issueLookup[ce.IssueID]
		}
		f.Errors.Append(e)
	}

	return f
}
