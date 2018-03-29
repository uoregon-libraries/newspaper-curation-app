// issue_errors.go centralizes all the ways an issue may need to report problems

package schema

import (
	"fmt"
	"path/filepath"
)

// IssueError implements apperr.Error and forms the base for all issue errors
type IssueError struct {
	Err  string
	Msg  string
	Prop bool
}

func (e *IssueError) Error() string {
	return e.Err
}

// Message returns the long, human-friendly error message
func (e *IssueError) Message() string {
	return e.Msg
}

// Propagate returns whether the error should flag the object's parent as also
// having an error
func (e *IssueError) Propagate() bool {
	return e.Prop
}

// errorIdent gives a consistent way to describe an issue which may not have a
// key that means the same thing as its actual location on disk
func (i *Issue) errorIdent() string {
	return fmt.Sprintf("%s issue %s/%s", i.WorkflowStep, i.Title.LCCN, filepath.Base(i.Location))
}

// ErrNoFiles adds an error stating the issue folder is empty
func (i *Issue) ErrNoFiles() {
	i.addError(&IssueError{
		Err:  "no files",
		Msg:  i.errorIdent() + " has no files",
		Prop: true,
	})
}

// ErrInvalidFolderName adds an Error for invalid folder name formats
func (i *Issue) ErrInvalidFolderName(extra string) {
	i.addError(&IssueError{
		Err:  "invalid folder name",
		Msg:  i.errorIdent() + " has an invalid folder name: " + extra,
		Prop: true,
	})
}

// ErrReadFailure indicates the issue's folder wasn't able to be read
func (i *Issue) ErrReadFailure(err error) {
	i.addError(&IssueError{
		Err:  err.Error(),
		Msg:  i.errorIdent() + " wasn't able to be scanned for files: " + err.Error(),
		Prop: true,
	})
}

// ErrFolderContents tells us the issue's files on disk are invalid in some way
func (i *Issue) ErrFolderContents(extra string) {
	i.addError(&IssueError{
		Err:  "missing / invalid folder contents",
		Msg:  i.errorIdent() + " doesn't have valid files: " + extra,
		Prop: true,
	})
}

// ErrTooNew adds an error for issues which are too new to be processed.  hours
// should be set to the minimum number of hours an issue should be untouched
// before being considered "safe".
func (i *Issue) ErrTooNew(hours int) {
	i.addError(&IssueError{
		Err:  "too new for processing",
		Msg:  fmt.Sprintf("%s must be left alone for a minimum of %d hours before processing", i.errorIdent(), hours),
		Prop: false,
	})
}

// DuplicateIssueError implements apperr.Error for duped issue situations, and
// holds onto extra information for figuring out how to handle the dupe
type DuplicateIssueError struct {
	*IssueError
	Location string
	Name     string
	IsLive   bool
}

// ErrDuped flags this issue with a DuplicateIssueError
func (i *Issue) ErrDuped(dupe *Issue) {
	i.addError(&DuplicateIssueError{
		IssueError: &IssueError{
			Err:  "duplicate of another issue",
			Msg:  fmt.Sprintf("%s is a likely duplicate of %s", i.errorIdent(), dupe.WorkflowIdentification()),
			Prop: true,
		},
		Location: dupe.Location,
		Name:     dupe.Title.Name + ", " + dupe.RawDate,
		IsLive:   dupe.WorkflowStep == WSInProduction,
	})
}
