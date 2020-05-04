package issuefinder

import (
	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// FindInProcessIssues aggregates all issues which have been indexed in the database
func (s *Searcher) FindInProcessIssues() error {
	s.init()

	var issues, err = models.FindInProcessIssues()
	if err != nil {
		return apperr.Errorf("unable to scan in-process issues from database: %s", err)
	}
	for _, issue := range issues {
		s.storeInProcessIssue(issue)
	}

	return nil
}

// storeInProcessIssue adds the issue's *schema* title to the searcher's title
// list, then adds the issue, converted to a schema issue, to the title.  We
// are losing information in the process, but this is just for indexing known
// issues, not manipulating them or linking them to anything else.
func (s *Searcher) storeInProcessIssue(dbIssue *models.Issue) {
	var title = s.findOrCreateDatabaseTitle(dbIssue)

	var issue, err = dbIssue.SchemaIssue()
	if err != nil {
		title.AddError(apperr.Errorf("invalid database issue id %d", dbIssue.ID))
		return
	}

	// TODO: Do other sanity checking as it makes sense

	title.AddIssue(issue)
	issue.FindFiles()
	s.Issues = append(s.Issues, issue)
}
