package presenter

import (
	"fmt"
	"html/template"
	"sftp"
	"web/webutil"
)

// Issue wraps an sftp Issue with presentation-specific logic
type Issue struct {
	*sftp.Issue
	Publisher *Publisher
	PDFs      []*PDF
}

// DecorateIssue returns a wrapped issue for the given publisher and sftp
// issue.  A decorated publisher is required so the decorated issue isn't
// re-decorating a publisher we've already wrapped.
func DecorateIssue(publisher *Publisher, si *sftp.Issue) *Issue {
	var issue = &Issue{Issue: si, Publisher: publisher}
	issue.buildPDFList()
	return issue
}

// buildPDFList stores the list of decorated PDFs from an issue's list of
// underlying sftp PDFs
func (issue *Issue) buildPDFList() {
	var sftpPDFs = issue.Issue.PDFs
	var list = make([]*PDF, len(sftpPDFs))
	for i, pdf := range sftpPDFs {
		list[i] = DecoratePDF(issue, pdf)
	}
	issue.PDFs = list
}

// Link returns the link to an issue's details page
func (issue *Issue) Link() template.HTML {
	var path = webutil.IssuePath(issue.Publisher.Name, issue.Name)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, issue.Name))
}
