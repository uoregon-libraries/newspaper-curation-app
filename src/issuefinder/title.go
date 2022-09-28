package issuefinder

import (
	"fmt"
	"path/filepath"

	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/chronam"
	"github.com/uoregon-libraries/newspaper-curation-app/src/httpcache"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// findOrCreateFilesystemTitle looks up the title by its given path and returns
// it or creates a new one if its "name" (last part of path) is in the
// database.  If a title still isn't found, one is created, but an error is
// attached to the searcher as we shouldn't be finding titles on the filesystem
// that aren't in the database.
func (s *Searcher) findOrCreateFilesystemTitle(path string) *schema.Title {
	var t *schema.Title
	var titleName = filepath.Base(path)
	if s.titleByLoc[path] == nil {
		t = s.dbTitles.Find(titleName).SchemaTitle()
		if t != nil {
			t.Location = path
			s.addTitle(t)
		}
	}

	// If we still have no title, we create one but make it clear it's a problem
	if t == nil {
		t = &schema.Title{LCCN: titleName, Location: path, Name: titleName}
		s.addTitle(t)
		t.AddError(apperr.Errorf("unable to find title %#v in database", titleName))
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
		return nil, fmt.Errorf("unable to GET %#v: %w", uri, err)
	}
	var tJSON *chronam.TitleJSON
	tJSON, err = chronam.ParseTitleJSON(contents)
	if err != nil {
		return nil, fmt.Errorf("unable to parse title JSON for %#v: %w", uri, err)
	}

	s.addTitle(&schema.Title{
		LCCN:               tJSON.LCCN,
		Name:               tJSON.Name,
		PlaceOfPublication: tJSON.PlaceOfPublication,
		Location:           uri,
	})
	return s.titleByLoc[uri], nil
}

// findOrCreateDatabaseTitle takes a database issue and returns the equivalent
// schema.Title stored in this searcher, or else looks up the issue's models.Title,
// creates an equivalent schema.Title and stores it, faking a location for
// future lookup
func (s *Searcher) findOrCreateDatabaseTitle(issue *models.Issue) *schema.Title {
	var t = s.dbTitles.Find(issue.LCCN)
	var fakeLocation string
	if t == nil {
		logger.Warnf("Invalid LCCN %q: no title in db matches this LCCN", issue.LCCN)
		fakeLocation = issue.LCCN
		var st = &schema.Title{Name: "**Unknown Title**", LCCN: issue.LCCN}
		st.AddError(apperr.New("No title with LCCN (or SFTP Directory) of %q exists"))
		s.titleByLoc[fakeLocation] = st
	} else {
		fakeLocation = t.LCCN
	}
	if s.titleByLoc[fakeLocation] == nil {
		var st = t.SchemaTitle()
		st.Location = fakeLocation
		s.addTitle(st)
	}

	return s.titleByLoc[fakeLocation]
}
