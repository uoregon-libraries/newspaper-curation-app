// issue_errors.go centralizes all the ways an issue may need to report problems

package schema

import (
	"fmt"

	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
)

// IssueError implements apperr.Error and forms the base for all issue errors
type IssueError struct {
	Err  string
	Msg  string
	Prop bool
	Warn bool
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

// Warning returns whether this error is classified low enough to allow other
// actions to happen
func (e *IssueError) Warning() bool {
	return e.Warn
}

// ErrNoFiles adds an error stating the issue folder is empty
func (i *Issue) ErrNoFiles() apperr.Error {
	return i.addError(&IssueError{
		Err:  "no files",
		Msg:  "Issue has no files",
		Prop: true,
	})
}

// ErrInvalidFolderName adds an Error for invalid folder name formats
func (i *Issue) ErrInvalidFolderName(extra string) apperr.Error {
	return i.addError(&IssueError{
		Err:  "invalid folder name",
		Msg:  "Issue has an invalid folder name: " + extra,
		Prop: true,
	})
}

// ErrReadFailure indicates the issue's folder wasn't able to be read
func (i *Issue) ErrReadFailure(err error) apperr.Error {
	return i.addError(&IssueError{
		Err:  err.Error(),
		Msg:  "Issue wasn't able to be scanned for files: " + err.Error(),
		Prop: true,
	})
}

// ErrFolderContents tells us the issue's files on disk are invalid in some way
func (i *Issue) ErrFolderContents(extra string) apperr.Error {
	return i.addError(&IssueError{
		Err:  "missing / invalid folder contents",
		Msg:  "Issue's folder contents are invalid: " + extra,
		Prop: true,
	})
}

// ErrTooNew adds an error for issues which are too new to be processed.  hours
// should be set to the minimum number of hours an issue should be untouched
// before being considered "safe".
func (i *Issue) ErrTooNew(hours int) apperr.Error {
	return i.addError(&IssueError{
		Err:  "too new for processing",
		Msg:  fmt.Sprintf("Issue was modified too recently; it must be left alone for a minimum of %d hours before processing", hours),
		Prop: false,
	})
}

// WarnTooNew sets a warning-level error for alerting curators without forcing
// the issue to be stuck
func (i *Issue) WarnTooNew() apperr.Error {
	return i.addError(&IssueError{
		Err:  "may be too new",
		Msg:  "Issue was modified recently and may still have updates pending",
		Prop: false,
		Warn: true,
	})
}

// DuplicateIssueError implements apperr.Error for duped issue situations, and
// holds onto extra information for figuring out how to handle the dupe
type DuplicateIssueError struct {
	*IssueError
	IssueID  int64
	Location string
	Name     string
	IsLive   bool
}

// ErrDuped flags this issue with a DuplicateIssueError
func (i *Issue) ErrDuped(dupe *Issue) apperr.Error {
	return i.addError(&DuplicateIssueError{
		IssueError: &IssueError{
			Err:  "duplicate of another issue",
			Msg:  fmt.Sprintf("This issue appears to be a duplicate (same LCCN, date, and edition) of %s", dupe.WorkflowIdentification()),
			Prop: true,
			Warn: true,
		},
		IssueID:  dupe.DatabaseID,
		Location: dupe.Location,
		Name:     dupe.Title.Name + ", " + dupe.RawDate,
		IsLive:   dupe.WorkflowStep == WSInProduction,
	})
}

// ErrBadTitle adds an error to the issue indicating that its title is invalid
// and therefore the issue cannot be processed even if all its data is good
func (i *Issue) ErrBadTitle() apperr.Error {
	return i.addError(&IssueError{
		Err:  "issue linked to invalid title",
		Msg:  "Title is invalid",
		Prop: false,
	})
}
