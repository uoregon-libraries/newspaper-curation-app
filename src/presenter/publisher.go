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
}

// DecoratePublisher wraps publisher and returns it
func DecoratePublisher(publisher *sftp.Publisher) *Publisher {
	return &Publisher{Publisher: publisher}
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

// IssueLink returns the link to an issue's details page
func (p *Publisher) IssueLink(name string) template.HTML {
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, webutil.IssuePath(p.Name, name), name))
}

// Show tells us whether this publisher should be displayed in the main list of
// publishers.  We specifically skip "publishers" with no issues, because
// they're sometimes new publishers we haven't fully set up, sometimes
// no-longer-participating publishers, and in all cases have no data to
// consider.
func (p *Publisher) Show() bool {
	return len(p.Issues) > 0
}
