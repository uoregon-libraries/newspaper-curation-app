package presenter

import (
	"sftp"
)

// Publisher wraps sftp.Publisher to provide presentation-specific functions
type Publisher struct {
	*sftp.Publisher
}

// PublisherList decorates a list of sftp publishers with presentation logic
// and returns it
func PublisherList(pubList []*sftp.Publisher) []*Publisher {
	var list = make([]*Publisher, len(pubList))
	for i, p := range pubList {
		list[i] = &Publisher{p}
	}

	return list
}

// Show tells us whether this publisher should be displayed in the main list of
// publishers.  We specifically skip "publishers" with no issues, because
// they're sometimes new publishers we haven't fully set up, sometimes
// no-longer-participating publishers, and in all cases have no data to
// consider.
func (p *Publisher) Show() bool {
	return len(p.Issues) > 0
}
