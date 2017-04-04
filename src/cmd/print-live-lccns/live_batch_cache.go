package main

import (
	"chronam"
	"fmt"
	"httpcache"
	"log"
	"net/url"
	"path"
)

var allTitles = make(map[string]*chronam.TitleJSON)

func cacheLiveTitles(hostname, cachePath string) error {
	var batchMetadataList, err = findAllLiveBatches(hostname, cachePath)
	if err != nil {
		return fmt.Errorf("unable to load batch list from %#v: %s", hostname, err)
	}

	// We (slightly) throttle JSON requests as there can be several hundred of these
	var c = httpcache.NewClient(cachePath, 50)
	for _, batchMetadata := range batchMetadataList {
		var lccns []string
		lccns, err = getBatchLCCNs(c, batchMetadata.URL)
		if err != nil {
			return fmt.Errorf("unable to load batch LCCNs for %#v: %s", batchMetadata.URL, err)
		}
		for _, lccn := range lccns {
			log.Printf("DEBUG - reading %#v title metadata", lccn)
			if allTitles[lccn] != nil {
				continue
			}
			var apiURL, err = url.Parse(hostname)
			if err != nil {
				return err
			}
			apiURL.Path = path.Join("lccn", lccn+".json")
			var request = httpcache.AutoRequest(apiURL.String(), "titles")
			var contents []byte
			contents, err = c.GetCachedBytes(request)
			if err != nil {
				return fmt.Errorf("Unable to GET %#v: %s", apiURL.String(), err)
			}
			var title *chronam.TitleJSON
			title, err = chronam.ParseTitleJSON(contents)
			if err != nil {
				return fmt.Errorf("Unable to parse title JSON for %#v: %s", apiURL.String(), err)
			}
			allTitles[lccn] = title
		}
	}

	return nil
}

func getBatchLCCNs(c *httpcache.Client, batchURL string) ([]string, error) {
	log.Printf("DEBUG - reading LCCNs for %#v", batchURL)
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

	return batch.LCCNs, nil
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
