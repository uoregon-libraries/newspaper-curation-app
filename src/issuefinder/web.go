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

// FindWebBatches reads through the JSON from the batch API URL and
// grabs "next" page until there is no next page.  Each batch is then read from
// the JSON cache path, or read from the site and cached.  The disk cache
// speeds up the tool's future runs by only having to request what's been
// batched since a prior run.
//
// As with other searches, this returns an error only on unexpected behaviors,
// like the site not responding.
func (f *Finder) FindWebBatches(hostname, cachePath string) error {
	var batchMetadataList, err = f.findAllLiveBatches(hostname, cachePath)
	if err != nil {
		return fmt.Errorf("unable to load batch list from %#v: %s", hostname, err)
	}

	// We (slightly) throttle batch JSON requests as there can be a few hundred of these
	var c = httpcache.NewClient(cachePath, 50)
	for _, batchMetadata := range batchMetadataList {
		var batch, err = schema.ParseBatchname(batchMetadata.Name)
		if err != nil {
			f.newError(hostname, fmt.Errorf("invalid live batch name %#v: %s", batchMetadata.Name, err))
			return nil
		}
		batch.Location = batchMetadata.URL
		f.Batches = append(f.Batches, batch)

		var issueMetadataList []*chronam.IssueMetadata
		issueMetadataList, err = f.findBatchedIssueMetadata(c, batchMetadata.URL)
		if err != nil {
			return fmt.Errorf("unable to load live issues from %#v: %s", batchMetadata.URL, err)
		}
		for _, meta := range issueMetadataList {
			var t, err = f.findLiveTitle(c, meta.Title.URL)
			if err != nil {
				return fmt.Errorf("unable to load live title %#v: %s", meta.Title.URL, err)
			}
			f.cacheLiveIssue(batch, t, meta)
		}
	}

	return nil
}

func (f *Finder) cacheLiveIssue(batch *schema.Batch, title *schema.Title, meta *chronam.IssueMetadata) {
	var dt, err = time.Parse("2006-01-02", meta.Date)
	if err != nil {
		f.newError(batch.Location, fmt.Errorf("invalid date for issue %#v: %s", meta, err)).SetBatch(batch)
		return
	}

	// We can determine edition from the issue URL, as it always ends in "ed-?.json"
	var base = path.Base(meta.URL)
	var editionString = base[3:]
	editionString = editionString[:len(editionString)-5]
	var edition int
	edition, err = strconv.Atoi(editionString)
	if err != nil {
		f.newError(batch.Location, fmt.Errorf("invalid edition for issue %#v", editionString, meta)).SetBatch(batch)
		return
	}

	var issue = &schema.Issue{Title: title, Date: dt, Edition: edition, Location: meta.URL, Batch: batch}
	f.Issues = append(f.Issues, issue)

	return
}

func (f *Finder) findBatchedIssueMetadata(c *httpcache.Client, batchURL string) ([]*chronam.IssueMetadata, error) {
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
func (f *Finder) findAllLiveBatches(hostname, cachePath string) ([]*chronam.BatchMetadata, error) {
	// We don't bother throttling because there won't be more than a handful of
	// batch list pages
	var c = httpcache.NewClient(cachePath, 0)

	var apiURL, err = url.Parse(hostname)
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

func (f *Finder) findLiveTitle(c *httpcache.Client, uri string) (*schema.Title, error) {
	// This is another horrible hack (title lookup needs fixing): we use the URI
	// for the title lookup because web titles shouldn't be looked up by peeling
	// apart the URL, but we need to avoid re-reading the same title data dozens
	// of times.  Even cached, that's just not smart.
	if f.titleLookup[uri] != nil {
		return f.titleLookup[uri], nil
	}

	var request = httpcache.AutoRequest(uri, "titles")
	var contents, err = c.GetCachedBytes(request)
	if err != nil {
		return nil, fmt.Errorf("unable to GET %#v: %s", uri, err)
	}
	var tJSON *chronam.TitleJSON
	tJSON, err = chronam.ParseTitleJSON(contents)
	if err != nil {
		return nil, fmt.Errorf("unable to parse title JSON for %#v: %s", uri, err)
	}

	// For now we just blow away whatever was there before, because live titles
	// are really quite separate from disk titles, and TODO they'll be split
	// apart very soon anyway
	var title = &schema.Title{LCCN: tJSON.LCCN}
	f.Titles = append(f.Titles, title)
	f.titleLookup[title.LCCN] = title
	f.titleLookup[uri] = title

	return title, nil
}
