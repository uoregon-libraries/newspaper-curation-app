package issuefinder

import (
	"db"
	"fmt"
)

// FindInProcessIssues aggregates all issues which have been indexed in the database
func (s *Searcher) FindInProcessIssues() error {
	s.init()

	var issues, err = db.FindInProcessIssues()
	if err != nil {
		return fmt.Errorf("unable to scan in-process issues from database: %s", err)
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
func (s *Searcher) storeInProcessIssue(dbIssue *db.Issue) {
	var title = s.findOrCreateDatabaseTitle(dbIssue)

	// We don't know the issue (or even if there is an issue object) yet, so we
	// need to aggregate errors.  And we shortcut the aggregation so we don't
	// forget to set the title.
	var errors []*Error
	var addErr = func(e error) { errors = append(errors, s.newError(dbIssue.Location, e).SetTitle(title)) }

	// Build the issue now that we know we can put together the minimal metadata
	var issue, err = dbIssue.SchemaIssue()
	if err != nil {
		addErr(err)
		return
	}

	// TODO: Do other sanity checking as it makes sense

	for _, e := range errors {
		e.SetIssue(issue)
	}

	title.AddIssue(issue)
	issue.FindFiles()
	s.Issues = append(s.Issues, issue)
}
