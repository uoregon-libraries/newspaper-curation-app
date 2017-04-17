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
	Title *Title
	PDFs  []*PDF
}

// DecorateIssue returns a wrapped issue for the given title and sftp
// issue.  A decorated title is required so the decorated issue isn't
// re-decorating a title we've already wrapped.
func DecorateIssue(title *Title, si *sftp.Issue) *Issue {
	var issue = &Issue{Issue: si, Title: title}
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
	var path = webutil.IssuePath(issue.Title.Name, issue.Name)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, issue.Name))
}
