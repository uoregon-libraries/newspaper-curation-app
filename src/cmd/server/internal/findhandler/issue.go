package findhandler

import (
	"schema"
)

// Issue wraps schema.Issue to provide presentation-specific information needed
// for the issue finder tool
type Issue struct {
	*schema.Issue
}

// DateString returns the issue date in the format we use for directory names
func (i *Issue) DateString() string {
	return i.Date.Format("2006-01-02")
}
