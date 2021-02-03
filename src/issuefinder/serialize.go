package issuefinder

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// cachedSearcher internally converts a Searcher into a cachedSearcher for serialization
func (s *Searcher) cachedSearcher() cachedSearcher {
	var cSrch = cachedSearcher{Namespace: s.Namespace, Location: s.Location}

	var batchIDLookup = make(map[*schema.Batch]cacheID)
	var batchLookup = make(map[cacheID]cachedBatch)
	var titleIDLookup = make(map[*schema.Title]cacheID)
	var titleLookup = make(map[cacheID]cachedTitle)
	var issueIDLookup = make(map[*schema.Issue]cacheID)
	var issueLookup = make(map[cacheID]cachedIssue)
	var fileIDLookup = make(map[*schema.File]cacheID)

	var issueID, titleID, batchID, fileID cacheID

	for _, b := range s.Batches {
		batchID++
		var cb = cachedBatch{
			ID:          batchID,
			MARCOrgCode: b.MARCOrgCode,
			Keyword:     b.Keyword,
			Version:     b.Version,
			Location:    b.Location,
			Errors:      b.Errors,
		}
		batchIDLookup[b] = batchID
		batchLookup[batchID] = cb
		cSrch.Batches = append(cSrch.Batches, cb)
	}
	for _, t := range s.Titles {
		titleID++
		var ct = cachedTitle{
			ID:                 titleID,
			LCCN:               t.LCCN,
			Name:               t.Name,
			PlaceOfPublication: t.PlaceOfPublication,
			Location:           t.Location,
			Errors:             t.Errors,
		}
		titleIDLookup[t] = titleID
		titleLookup[titleID] = ct
		cSrch.Titles = append(cSrch.Titles, ct)
	}
	for _, i := range s.Issues {
		issueID++
		var ci = cachedIssue{
			ID:           issueID,
			RawDate:      i.RawDate,
			Edition:      i.Edition,
			Location:     i.Location,
			WorkflowStep: string(i.WorkflowStep),
			Errors:       i.Errors,
		}
		for _, f := range i.Files {
			fileID++
			var cf = cachedFile{
				ID:       fileID,
				File:     *f.File,
				Location: f.Location,
				Errors:   f.Errors,
			}
			fileIDLookup[f] = fileID
			ci.Files = append(ci.Files, cf)
		}
		issueIDLookup[i] = issueID
		issueLookup[issueID] = ci
		ci.TitleID = titleIDLookup[i.Title]

		if i.Batch != nil {
			ci.BatchID = batchIDLookup[i.Batch]
		}

		cSrch.Issues = append(cSrch.Issues, ci)
	}

	cSrch.Errors = s.Errors
	return cSrch
}

// cachedFinder iterates over the searchers to create a serializable cachedFinder
func (f *Finder) cachedFinder() cachedFinder {
	var cFind cachedFinder
	for _, srch := range f.Searchers {
		cFind.Searchers = append(cFind.Searchers, srch.cachedSearcher())
	}
	return cFind
}

// Serialize writes the Finder's state to the given filename or returns an error
func (f *Finder) Serialize(outFilename string) error {
	// Set up a temp file to store the serialization so we aren't writing to a
	// file which may have valid data in it already
	var tmpfile, err = fileutil.TempFile("", "finder-serialize-", "")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())

	// Register all the error types
	gob.Register(&apperr.BaseError{})
	gob.Register(&schema.IssueError{})
	gob.Register(&schema.DuplicateIssueError{})

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
	err = fileutil.CopyFile(tmpfile.Name(), outFilename)
	if err != nil {
		return fmt.Errorf("unable to copy temp file to %q: %s", outFilename, err)
	}

	// Attempt to remove the backup, though we ignore any errors if it doesn't
	// work; we don't want to fail the whole operation because a backup file got
	// left behind, do we?
	if backup != "" {
		os.Remove(backup)
	}
	return nil
}
