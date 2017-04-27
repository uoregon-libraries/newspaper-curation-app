package issuefinder

import (
	"fmt"
	"schema"
	"strings"
)

// ErrorList just stores the full error list and adds some functionality for
// easier lookup by various objects errors can be tied to
type ErrorList struct {
	Errors []*Error

	// These error maps let us look up errors by their "primary" object - that
	// is, the lowest-level item associated with the error

	// IssueErrors stores all errors for a unique Issue.  Note that issues are
	// completely unique per location: two issues for the same date and LCCN in
	// different locations are separate objects.
	IssueErrors map[*schema.Issue][]*Error

	// TitleErrors stores errors for a Title.  Titles are unique per location.
	TitleErrors map[*schema.Title][]*Error

	// BatchErrors stores errors for a batch.  Batches are unique per location.
	BatchErrors map[*schema.Batch][]*Error

	// OtherErrors gets everything not assigned to an object
	OtherErrors []*Error
}

// Append simply adds the given error to the raw list
func (list *ErrorList) Append(e *Error) {
	list.Errors = append(list.Errors, e)
}

// Index should be run after all errors are appended in order to use the lookup
// functions.  If it isn't run, the lookups won't have data and will just
// return nil.
func (list *ErrorList) Index() {
	// Initialize / reset maps and array
	list.IssueErrors = make(map[*schema.Issue][]*Error)
	list.TitleErrors = make(map[*schema.Title][]*Error)
	list.BatchErrors = make(map[*schema.Batch][]*Error)
	list.OtherErrors = make([]*Error, 0)

	for _, e := range list.Errors {
		if e.Issue != nil {
			list.IssueErrors[e.Issue] = append(list.IssueErrors[e.Issue], e)
		} else if e.Title != nil {
			list.TitleErrors[e.Title] = append(list.TitleErrors[e.Title], e)
		} else if e.Batch != nil {
			list.BatchErrors[e.Batch] = append(list.BatchErrors[e.Batch], e)
		} else {
			list.OtherErrors = append(list.OtherErrors, e)
		}
	}
}

// Error combines the standard error interface with some context so we can
// easily categorize errors.  Error is built somewhat functionally in order to
// more easily chain together calls:
//
//     var err = fmt.Errorf("invalid issue directory name %q", issuePath)
//     finder.newError(path, err).SetBatch(batch).SetTitle(title)
type Error struct {
	Batch    *schema.Batch
	Title    *schema.Title
	Issue    *schema.Issue
	File     *schema.File
	Location string
	Error    error
}

// newError creates an Error with the two required pieces of information:
// location and underlying error interface.  This is private on purpose;
// nothing external should be creating issuefinder errors.
func (f *Finder) newError(loc string, err error) *Error {
	var e = &Error{Location: loc, Error: err}
	f.Errors.Append(e)
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

// SetIssue changes the issue, title, and batch, and returns the error
func (e *Error) SetIssue(i *schema.Issue) *Error {
	e.Issue = i
	e.Title = i.Title
	e.Batch = i.Batch
	return e
}

// SetFile changes the file, issue, title, and batch, and returns the error
func (e *Error) SetFile(f *schema.File) *Error {
	e.SetIssue(f.Issue)
	e.File = f
	return e
}

// Message returns a description of the error and all the error's context
func (e *Error) Message() string {
	var details []string
	if e.Issue != nil {
		details = append(details, e.Issue.Key())
	}
	if e.Batch != nil {
		details = append(details, fmt.Sprintf("%s", e.Batch.Fullname()))
	}

	details = append(details, e.Location)
	var msg = strings.Join(details, "; ")
	return fmt.Sprintf("%s (%s)", e.Error, msg)
}
