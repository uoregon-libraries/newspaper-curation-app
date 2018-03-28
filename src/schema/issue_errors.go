// issue_errors.go centralizes all the ways an issue may need to report problems

package schema

import (
	"fmt"
	"path/filepath"
)

// issueError implements apperr.Error and forms the base for all issue errors
type issueError struct {
	i    *Issue
	err  string
	msg  string
	prop bool
}

func (e *issueError) Error() string {
	return e.err
}

func (e *issueError) Message() string {
	return e.msg
}

func (e *issueError) Propagate() bool {
	return e.prop
}

// errorIdent gives a consistent way to describe an issue which may not have a
// key that means the same thing as its actual location on disk
func (i *Issue) errorIdent() string {
	return fmt.Sprintf("%s issue %s/%s", i.WorkflowStep, i.Title.LCCN, filepath.Base(i.Location))
}

// ErrNoFiles adds an error stating the issue folder is empty
func (i *Issue) ErrNoFiles() {
	i.addError(&issueError{
		i:    i,
		err:  "no files",
		msg:  i.errorIdent() + " has no files",
		prop: true,
	})
}

// ErrInvalidFolderName adds an Error for invalid folder name formats
func (i *Issue) ErrInvalidFolderName(extra string) {
	i.addError(&issueError{
		i:    i,
		err:  "invalid folder name",
		msg:  i.errorIdent() + " has an invalid folder name: " + extra,
		prop: true,
	})
}

// ErrReadFailure indicates the issue's folder wasn't able to be read
func (i *Issue) ErrReadFailure(err error) {
	i.addError(&issueError{
		i:    i,
		err:  err.Error(),
		msg:  i.errorIdent() + " wasn't able to be scanned for files: " + err.Error(),
		prop: true,
	})
}

// ErrFolderContents tells us the issue's files on disk are invalid in some way
func (i *Issue) ErrFolderContents(extra string) {
	i.addError(&issueError{
		i:    i,
		err:  "missing / invalid folder contents",
		msg:  i.errorIdent() + " doesn't have valid files: " + extra,
		prop: true,
	})
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
