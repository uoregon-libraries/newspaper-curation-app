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

// finder iterates over the cachedFinder's searchers and puts their data into a
// Finder
func (cf cachedFinder) finder() *Finder {
	var f = New()
	for _, cSrch := range cf.Searchers {
		var srch = cSrch.searcher()
		f.storeSearcher(srch)
	}
	return f
}

// searcher internally converts the cache-friendly data to a Searcher instance
func (cs cachedSearcher) searcher() *Searcher {
	var batchLookup = make(map[cacheID]*schema.Batch)
	var titleLookup = make(map[cacheID]*schema.Title)
	var issueLookup = make(map[cacheID]*schema.Issue)
	var fileLookup = make(map[cacheID]*schema.File)
	var srch = NewSearcher(cs.Namespace, cs.Location)

	// Build the basic schema objects with associations
	for _, cb := range cs.Batches {
		var b = &schema.Batch{
			MARCOrgCode: cb.MARCOrgCode,
			Keyword:     cb.Keyword,
			Version:     cb.Version,
			Location:    cb.Location,
		}
		batchLookup[cb.ID] = b
		srch.Batches = append(srch.Batches, b)
	}
	for _, ct := range cs.Titles {
		var t = &schema.Title{
			LCCN:               ct.LCCN,
			Name:               ct.Name,
			PlaceOfPublication: ct.PlaceOfPublication,
			Location:           ct.Location,
		}
		titleLookup[ct.ID] = t
		srch.Titles = append(srch.Titles, t)
		srch.titleByLoc[t.Location] = t
	}
	for _, ci := range cs.Issues {
		var i = &schema.Issue{
			Date:     ci.Date,
			Edition:  ci.Edition,
			Location: ci.Location,
		}
		for _, cf := range ci.Files {
			// Copy the fileutil.File structure or we get reused data
			var dupedFile = cf.File
			var file = &schema.File{File: &dupedFile, Location: cf.Location, Issue: i}
			fileLookup[cf.ID] = file
			i.Files = append(i.Files, file)
		}
		issueLookup[ci.ID] = i
		srch.Issues = append(srch.Issues, i)

		// Associate the title and batch; batch is optional, but title isn't
		if ci.BatchID != 0 {
			batchLookup[ci.BatchID].AddIssue(i)
		}
		titleLookup[ci.TitleID].AddIssue(i)
	}

	// Populate the Errors list
	for _, ce := range cs.Errors {
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
		if ce.FileID != 0 {
			e.File = fileLookup[ce.FileID]
		}
		srch.Errors.Append(e)
	}

	return srch
}
