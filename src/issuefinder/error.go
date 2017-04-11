package issuefinder

import (
	"schema"
)

// Error combines the standard error interface with some context so we can
// easily categorize errors.  Error is built somewhat functionally in order to
// more easily chain together calls:
//
//     finder.newError(path, fmt.Errorf("invalid issue directory name")).SetBatch(batch).SetTitle(title)
type Error struct {
	Batch    *schema.Batch
	Title    *schema.Title
	Issue    *schema.Issue
	Location string
	Error    error
}

// newError creates an Error with the two required pieces of information:
// location and underlying error interface.  This is private on purpose;
// nothing external should be creating issuefinder errors.
func (f *Finder) newError(loc string, err error) *Error {
	var e = &Error{Location: loc, Error: err}
	f.Errors = append(f.Errors, e)
	return e
}

// SetBatch changes the batch and returns the Error
func (e *Error) SetBatch(b *schema.Batch) *Error {
	e.Batch = b
	return e
}

// SetTitle changes the title and returns the error
func (e *Error) SetTitle(t *schema.Title) *Error {
	e.Title = t
	return e
}

// SetIssue changes the issue and returns the error
func (e *Error) SetIssue(i *schema.Issue) *Error {
	e.Issue = i
	return e
}
