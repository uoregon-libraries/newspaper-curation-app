package issuewatcher

import (
	"issuefinder"
	"issuesearch"
	"path/filepath"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

// CacheFile returns the standard path to the cache file based on the
// configuration of the watcher
func (w *Watcher) CacheFile() string {
	return filepath.Join(w.tempdir, "finder.cache")
}

// Serialize writes all internal search data to the CacheFile
func (w *Watcher) Serialize() error {
	return w.finder.Serialize(w.CacheFile())
}

// Deserialize attempts to read the CacheFile if it exists, populating the
// searchers and issue lookup
func (w *Watcher) Deserialize() error {
	var cacheFile = w.CacheFile()
	if fileutil.Exists(cacheFile) {
		var finder, err = issuefinder.Deserialize(cacheFile)
		if err != nil {
			return err
		}
		w.finder = finder
		w.lookup = issuesearch.NewLookup()
		w.lookup.Populate(w.finder.Issues)
	}
	return nil
}
