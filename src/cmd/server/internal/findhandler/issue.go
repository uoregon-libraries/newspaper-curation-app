package findhandler

import (
	"fmt"
	"schema"
)

// Issue wraps schema.Issue to provide presentation-specific information needed
// for the issue finder tool
type Issue struct {
	*schema.Issue
}

// DateEdition returns the issue date and edition in a user-friendly way
func (i *Issue) DateEdition() string {
	return fmt.Sprintf("%s, ed. %d", i.Date.Format("2006-01-02"), i.Edition)
}
