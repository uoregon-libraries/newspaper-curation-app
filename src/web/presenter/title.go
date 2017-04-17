package presenter

import (
	"fmt"
	"html/template"
	"sftp"
	"web/webutil"
)

// Title wraps sftp.Title to provide presentation-specific functions
type Title struct {
	*sftp.Title
	Issues []*Issue
}

// DecorateTitle wraps title and returns it
func DecorateTitle(title *sftp.Title) *Title {
	var t = &Title{Title: title}
	t.buildIssueList()
	return t
}

// TitleList decorates a list of sftp titles with presentation logic
// and returns it
func TitleList(tList []*sftp.Title) []*Title {
	var list = make([]*Title, len(tList))
	for i, p := range tList {
		list[i] = DecorateTitle(p)
	}

	return list
}

// Link returns a link to a given title's details page
func (t *Title) Link() template.HTML {
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, webutil.TitlePath(t.Name), t.Name))
}

// buildIssueList stores the list of decorated issues from a title's list
// of underlying sftp issues
func (t *Title) buildIssueList() {
	var sftpIssues = t.Title.Issues
	var list = make([]*Issue, len(sftpIssues))
	for i, issue := range sftpIssues {
		list[i] = DecorateIssue(t, issue)
	}
	t.Issues = list
}

// Show tells us whether this title should be displayed in the main list of
// titles.  We specifically skip "titles" with no issues, because
// they're sometimes new titles we haven't fully set up, sometimes
// no-longer-participating titles, and in all cases have no data to
// consider.
func (t *Title) Show() bool {
	return len(t.Issues) > 0
}
