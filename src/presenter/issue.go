package presenter

import (
	"fmt"
	"html/template"
	"sftp"
	"webutil"
)

// Issue wraps an sftp Issue with presentation-specific logic
type Issue struct {
	*sftp.Issue
	Publisher *Publisher
}

// DecorateIssue returns a wrapped issue for the given publisher and sftp
// issue.  A decorated publisher is required so the decorated issue isn't
// re-decorating a publisher we've already wrapped.
func DecorateIssue(publisher *Publisher, issue *sftp.Issue) *Issue {
	return &Issue{Issue: issue, Publisher: publisher}
}

// Link returns the link to an issue's details page
func (issue *Issue) Link() template.HTML {
	var path = webutil.IssuePath(issue.Publisher.Name, issue.Name)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, issue.Name))
}

