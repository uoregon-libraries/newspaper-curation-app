// issue_errors.go centralizes all the ways an issue may need to report problems

package schema

import (
	"fmt"
	"path/filepath"
)

// errorIdent gives a consistent way to describe an issue which may not have a
// key that means the same thing as its actual location on disk
func (i *Issue) errorIdent() string {
	return fmt.Sprintf("%s issue %s/%s", i.WorkflowStep, i.Title.LCCN, filepath.Base(i.Location))
}

// DuplicateIssueError implements apperr.Error for duped issue situations, and
// holds onto extra information for figuring out how to handle the dupe
type DuplicateIssueError struct {
	Issue *Issue
	Dupe  *Issue
}

// Error returns the simple explanation: this issue is duplicated somewhere
func (e *DuplicateIssueError) Error() string {
	return "duplicate of another issue"
}

// Message returns more detailed information about the duplicate
func (e *DuplicateIssueError) Message() string {
	return fmt.Sprintf("%s is a likely duplicate of %s", e.Issue.errorIdent(), e.Dupe.WorkflowIdentification())
}

// Propagate returns true for duped issues, as these errors are fairly severe
func (e *DuplicateIssueError) Propagate() bool {
	return true
}

// ErrDuped flags this issue with a DuplicateIssueError
func (i *Issue) ErrDuped(dupe *Issue) {
	i.addError(&DuplicateIssueError{Issue: i, Dupe: dupe})
}
