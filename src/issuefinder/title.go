package issuefinder

import (
	"chronam"
	"db"
	"fmt"
	"httpcache"
	"schema"
)

// findFilesystemTitle looks up the title by its given path and returns it or
// creates a new one if its "titleName" is in the database.  "titleName" can
// be LCCN or SFTP directory depending on the type of directory.
func (s *Searcher) findFilesystemTitle(titleName, path string) *schema.Title {
	if s.titleByLoc[path] == nil {
		// Make sure titles are loaded from the DB, and puke on any errors
		var err = db.LoadTitles()
		if err != nil {
			panic(err)
		}
		var t = db.LookupTitle(titleName).SchemaTitle()
		if t == nil {
			return nil
		}
		t.Location = path
		s.addTitle(t)
	}
	return s.titleByLoc[path]
}

// findOrCreateUnknownFilesystemTitle looks up the title by path and returns it
// or creates a new one.  This should only be used for titles for which we have
// no metadata: when LCCN is the only data available, the title is incomplete.
func (s *Searcher) findOrCreateUnknownFilesystemTitle(lccn, path string) *schema.Title {
	// First see if we can look up the title in the database
	if s.titleByLoc[path] == nil {
		s.findFilesystemTitle(lccn, path)
	}
	// If it's still empty, we create it with the limited data we have
	if s.titleByLoc[path] == nil {
		s.addTitle(&schema.Title{LCCN: lccn, Location: path})
	}
	return s.titleByLoc[path]
}

// addTitle pushes the title into the global titles list and caches it by its
// location field
func (s *Searcher) addTitle(title *schema.Title) {
	s.Titles = append(s.Titles, title)
	s.titleByLoc[title.Location] = title
}

// findOrCreateWebTitle looks up the title by its given URI and returns it or
// requests the URI to create, cache, and return a new one
func (s *Searcher) findOrCreateWebTitle(c *httpcache.Client, uri string) (*schema.Title, error) {
	if s.titleByLoc[uri] != nil {
		return s.titleByLoc[uri], nil
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

	s.addTitle(&schema.Title{
		LCCN:               tJSON.LCCN,
		Name:               tJSON.Name,
		PlaceOfPublication: tJSON.PlaceOfPublication,
		Location:           uri,
	})
	return s.titleByLoc[uri], nil
}
