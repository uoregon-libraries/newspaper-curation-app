package issuefinder

import (
	"chronam"
	"fmt"
	"httpcache"
	"net/url"
	"path"
	"schema"
	"strconv"
	"time"
)

// FindWebBatches reads through the JSON from the batch API URL (using
// Searcher's Location as the web root) and grabs "next" page until there is no
// next page.  Each batch is then read from the JSON cache path, or read from
// the site and cached.  The disk cache speeds up the tool's future runs by
// only having to request what's been batched since a prior run.
//
// As with other searches, this returns an error only on unexpected behaviors,
// like the site not responding.
func (s *Searcher) FindWebBatches(cachePath string) error {
	s.init()

	var batchMetadataList, err = s.findAllLiveBatches(cachePath)
	if err != nil {
		return fmt.Errorf("unable to load batch list from %#v: %s", s.Location, err)
	}

	// We (slightly) throttle batch JSON requests as there can be a few hundred of these
	var c = httpcache.NewClient(cachePath, 50)
	for _, batchMetadata := range batchMetadataList {
		var batch, err = schema.ParseBatchname(batchMetadata.Name)
		if err != nil {
			s.newError(s.Location, fmt.Errorf("invalid live batch name %#v: %s", batchMetadata.Name, err))
			return nil
		}
		batch.Location = batchMetadata.URL
		s.Batches = append(s.Batches, batch)

		var issueMetadataList []*chronam.IssueMetadata
		issueMetadataList, err = s.findBatchedIssueMetadata(c, batchMetadata.URL)
		if err != nil {
			return fmt.Errorf("unable to load live issues from %#v: %s", batchMetadata.URL, err)
		}
		for _, meta := range issueMetadataList {
			var t, err = s.findOrCreateWebTitle(c, meta.Title.URL)
			if err != nil {
				return fmt.Errorf("unable to load live title %#v: %s", meta.Title.URL, err)
			}
			s.cacheLiveIssue(batch, t, meta)
		}
	}

	return nil
}

func (s *Searcher) cacheLiveIssue(batch *schema.Batch, title *schema.Title, meta *chronam.IssueMetadata) {
	var _, err = time.Parse("2006-01-02", meta.Date)
	if err != nil {
		s.newError(batch.Location, fmt.Errorf("invalid date for issue %#v: %s", meta, err)).SetBatch(batch)
		return
	}

	// We can determine edition from the issue URL, as it always ends in "ed-?.json"
	var base = path.Base(meta.URL)
	var editionString = base[3:]
	editionString = editionString[:len(editionString)-5]
	var edition int
	edition, err = strconv.Atoi(editionString)
	if err != nil {
		s.newError(batch.Location, fmt.Errorf("invalid edition (%#v) for issue %#v", editionString, meta)).SetBatch(batch)
		return
	}

	var issue = &schema.Issue{RawDate: meta.Date, Edition: edition, Location: meta.URL, WorkflowStep: schema.WSInProduction}
	title.AddIssue(issue)
	batch.AddIssue(issue)
	s.Issues = append(s.Issues, issue)

	return
}

func (s *Searcher) findBatchedIssueMetadata(c *httpcache.Client, batchURL string) ([]*chronam.IssueMetadata, error) {
	var request = httpcache.AutoRequest(batchURL, "batches")
	var contents, err = c.GetCachedBytes(request)
	if err != nil {
		return nil, fmt.Errorf("unable to GET %#v: %s", batchURL, err)
	}

	var batch *chronam.BatchJSON
	batch, err = chronam.ParseBatchJSON(contents)
	if err != nil {
		return nil, fmt.Errorf("invalid JSON in %#v: %s", batchURL, err)
	}

	return batch.Issues, nil
}

// findAllLiveBatches hits the web server to request the full list of every
// known batch.  Results are stored in the cache path.  The returned structures
// are the aggregated batch metadata objects found after traversing all pages
// of batches.
func (s *Searcher) findAllLiveBatches(cachePath string) ([]*chronam.BatchMetadata, error) {
	// We don't bother throttling because there won't be more than a handful of
	// batch list pages
	var c = httpcache.NewClient(cachePath, 0)

	var apiURL, err = url.Parse(s.Location)
	if err != nil {
		return nil, err
	}
	apiURL.Path = "batches.json"
	var batchList = &chronam.BatchesListJSON{Next: apiURL.String()}

	var page int
	var batchMetadataList []*chronam.BatchMetadata
	var request = &httpcache.Request{Subdirectory: "batch-list", Extension: "json"}

	for batchList.Next != "" {
		page++
		request.Filename = fmt.Sprintf("page-%d", page)
		request.URL = batchList.Next

		var contents, err = c.ForceGetBytes(request)
		if err != nil {
			return nil, err
		}

		// Create and deserialize into a new structure
		batchList, err = chronam.ParseBatchesListJSON(contents)
		if err != nil {
			return nil, err
		}
		for _, b := range batchList.Batches {
			batchMetadataList = append(batchMetadataList, b)
		}
	}

	return batchMetadataList, nil
}
