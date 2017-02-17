package presenter

import (
	"fmt"
	"html/template"
	"sftp"
	"webutil"
)

// Publisher wraps sftp.Publisher to provide presentation-specific functions
type Publisher struct {
	*sftp.Publisher
	Issues []*Issue
}

// DecoratePublisher wraps publisher and returns it
func DecoratePublisher(publisher *sftp.Publisher) *Publisher {
	var p = &Publisher{Publisher: publisher}
	p.buildIssueList()
	return p
}

// PublisherList decorates a list of sftp publishers with presentation logic
// and returns it
func PublisherList(pubList []*sftp.Publisher) []*Publisher {
	var list = make([]*Publisher, len(pubList))
	for i, p := range pubList {
		list[i] = DecoratePublisher(p)
	}

	return list
}

// Link returns a link to a given publisher's details page
func (p *Publisher) Link() template.HTML {
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, webutil.PublisherPath(p.Name), p.Name))
}

// buildIssueList stores the list of decorated issues from a publisher's list
// of underlying sftp issues
func (p *Publisher) buildIssueList() {
	var sftpIssues = p.Publisher.Issues
	var list = make([]*Issue, len(sftpIssues))
	for i, issue := range sftpIssues {
		list[i] = DecorateIssue(p, issue)
	}
	p.Issues = list
}

// Show tells us whether this publisher should be displayed in the main list of
// publishers.  We specifically skip "publishers" with no issues, because
// they're sometimes new publishers we haven't fully set up, sometimes
// no-longer-participating publishers, and in all cases have no data to
// consider.
func (p *Publisher) Show() bool {
	return len(p.Issues) > 0
}
