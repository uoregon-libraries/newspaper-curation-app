package issuefinder

import (
	"encoding/gob"
	"fileutil"
	"fmt"
	"io/ioutil"
	"os"
	"schema"
)

// cachedFinder internally converts a Finder into a cachedFinder for Finder.Serialize
func (f *Finder) cachedFinder() cachedFinder {
	var cf cachedFinder
	var batchIDLookup = make(map[*schema.Batch]cacheID)
	var batchLookup = make(map[cacheID]cachedBatch)
	var titleIDLookup = make(map[*schema.Title]cacheID)
	var titleLookup = make(map[cacheID]cachedTitle)
	var issueIDLookup = make(map[*schema.Issue]cacheID)
	var issueLookup = make(map[cacheID]cachedIssue)

	var issueID, titleID, batchID cacheID

	for _, i := range f.Issues {
		issueID++
		var ci = cachedIssue{ID: issueID, Date: i.Date, Edition: i.Edition, Location: i.Location}
		issueIDLookup[i] = issueID
		issueLookup[issueID] = ci

		if titleIDLookup[i.Title] == 0 {
			titleID++
			var ct = cachedTitle{
				ID:                 titleID,
				LCCN:               i.Title.LCCN,
				Name:               i.Title.Name,
				PlaceOfPublication: i.Title.PlaceOfPublication,
				Location:           i.Title.Location,
			}
			titleIDLookup[i.Title] = titleID
			titleLookup[titleID] = ct
			cf.Titles = append(cf.Titles, ct)
		}
		ci.TitleID = titleIDLookup[i.Title]

		if i.Batch != nil {
			if batchIDLookup[i.Batch] == 0 {
				batchID++
				var cb = cachedBatch{
					ID:          batchID,
					MARCOrgCode: i.Batch.MARCOrgCode,
					Keyword:     i.Batch.Keyword,
					Version:     i.Batch.Version,
					Location:    i.Batch.Location,
				}
				batchIDLookup[i.Batch] = batchID
				batchLookup[batchID] = cb
				cf.Batches = append(cf.Batches, cb)
			}
			ci.BatchID = batchIDLookup[i.Batch]
		}

		cf.Issues = append(cf.Issues, ci)
	}

	for _, e := range f.Errors.Errors {
		var b = e.Batch
		var t = e.Title
		var i = e.Issue

		var ce = cachedError{Location: e.Location, Error: e.Error.Error()}
		if b != nil {
			ce.BatchID = batchIDLookup[b]
		}
		if t != nil {
			ce.TitleID = titleIDLookup[t]
		}
		if i != nil {
			ce.IssueID = issueIDLookup[i]
		}
		cf.Errors = append(cf.Errors, ce)
	}

	return cf
}

// Serialize writes the Finder's state to the given filename or returns an error
func (f *Finder) Serialize(outFilename string) error {
	// Set up a temp file to store the serialization so we aren't writing to a
	// file which may have valid data in it already
	var tmpfile, err = ioutil.TempFile("", "finder-serialize-")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())

	// Attempt to encode to said file, returning the error if that doesn't work
	var enc = gob.NewEncoder(tmpfile)
	err = enc.Encode(f.cachedFinder())
	if err != nil {
		return err
	}

	// Continue the paranoia: if the file exists, we make a backup instead of
	// just overwriting it
	var backup string
	if fileutil.Exists(outFilename) {
		backup = tmpfile.Name() + "-bak"
		err = fileutil.CopyFile(outFilename, backup)
		if err != nil {
			return fmt.Errorf("unable to backup original file %#v: %s", outFilename, err)
		}
	}

	// Create/overwrite the real file
	fileutil.CopyFile(tmpfile.Name(), outFilename)

	// Attempt to remove the backup, though we ignore any errors if it doesn't
	// work; we don't want to fail the whole operation because a backup file got
	// left behind, do we?
	if backup != "" {
		os.Remove(backup)
	}
	return nil
}
