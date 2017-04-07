package main

import (
	"chronam"
	"fmt"
	"httpcache"
	"log"
	"net/url"
	"path"
	"strconv"
	"time"
)

// cacheLiveBatchedIssues reads through the JSON from the batch API URL and
// grabs "next" page until there is no next page.  Each batch is then read from
// the JSON cache path, or read from the site and cached.  The disk cache
// speeds up the tool's future runs by only having to request what's been
// batched since a prior run.
func cacheLiveBatchedIssues(hostname, cachePath string) error {
	var batchMetadataList, err = findAllLiveBatches(hostname, cachePath)
	if err != nil {
		return fmt.Errorf("unable to load batch list from %#v: %s", hostname, err)
	}

	// We (slightly) throttle batch JSON requests as there can be a few hundred of these
	var c = httpcache.NewClient(cachePath, 50)
	for _, batchMetadata := range batchMetadataList {
		var batch, err = ParseBatchname(batchMetadata.Name)
		if err != nil {
			return fmt.Errorf("invalid live batch name %#v: %s", batchMetadata.Name, err)
		}

		var issueMetadataList []*chronam.IssueMetadata
		issueMetadataList, err = findBatchedIssueMetadata(c, batchMetadata.URL)
		if err != nil {
			return fmt.Errorf("unable to load live issues from %#v: %s", batchMetadata.URL, err)
		}
		err = cacheLiveIssuesFromMetadata(batch, issueMetadataList)
		if err != nil {
			return fmt.Errorf("unable to load live issues from %#v: %s", batchMetadata.URL, err)
		}
	}

	return nil
}

func cacheLiveIssuesFromMetadata(batch *Batch, issueMetadataList []*chronam.IssueMetadata) error {
	for _, meta := range issueMetadataList {
		var title = getTitleFromIssueMetadata(meta)
		var dt, err = time.Parse("2006-01-02", meta.Date)
		if err != nil {
			return fmt.Errorf("invalid date for issue %#v: %s", meta, err)
		}

		// We can determine edition from the issue URL, as it always ends in "ed-?.json"
		var base = path.Base(meta.URL)
		var editionString = base[3:]
		editionString = editionString[:len(editionString)-5]
		var edition int
		edition, err = strconv.Atoi(editionString)
		if err != nil {
			return fmt.Errorf("invalid edition (%#v) for issue %#v", editionString, meta)
		}

		var issue = title.AppendIssue(dt, edition)
		issue.Batch = batch
		cacheWebIssue(issue, meta.URL)
	}

	return nil
}

// getTitleFromIssueMetadata finds or creates a title from the given issue
// metadata.  If a new title is created, it's stored in the title lookup.
func getTitleFromIssueMetadata(meta *chronam.IssueMetadata) *Title {
	var base = path.Base(meta.Title.URL)
	var titleLCCN = base[:len(base)-5]
	return findOrCreateTitle(titleLCCN)
}

func findBatchedIssueMetadata(c *httpcache.Client, batchURL string) ([]*chronam.IssueMetadata, error) {
	log.Printf("DEBUG - reading issues list for %#v", batchURL)
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
func findAllLiveBatches(hostname, cachePath string) ([]*chronam.BatchMetadata, error) {
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

		log.Printf("DEBUG - reading batches list page %d: %#v", page, request.URL)
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
