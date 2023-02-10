package issuewatcher

import (
	"path/filepath"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/issuefinder"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// CacheFile returns the standard path to the cache file based on the
// configuration of the watcher
func (s *Scanner) CacheFile() string {
	return filepath.Join(s.Tempdir, "finder.cache")
}

// Serialize writes all internal search data to the CacheFile
func (s *Scanner) Serialize() error {
	return s.Finder.Serialize(s.CacheFile())
}

// Deserialize attempts to read the CacheFile if it exists, populating the
// searchers and issue lookup
func (s *Scanner) Deserialize() error {
	var cacheFile = s.CacheFile()
	if fileutil.Exists(cacheFile) {
		var finder, err = issuefinder.Deserialize(cacheFile)
		if err != nil {
			return err
		}
		s.Finder = finder
		s.Lookup = schema.NewLookup()
		err = s.Lookup.Populate(s.Finder.Issues)
		if err != nil {
			return err
		}
	}
	return nil
}
