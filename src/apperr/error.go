package apperr

import "fmt"

// Error is a simple error implementation meant to be serializable so we can
// store errors on batches, titles, issues, and files in the same way we can
// return them from various operations.  Our interface also provides for a
// Message() function which can be used for more human-friendly output for
// errors which are user-facing as well as functions which can be used to
// generally determine how to handle the error.
type Error interface {
	Error() string
	Message() string
	Propagate() bool // Not all errors should flag the parent as having errors
	Warning() bool   // If true, this is just a warning and other actions (like queueing uploaded issues) can continue
}

// BaseError implements Error
type BaseError struct {
	ErrorString string
}

func (e BaseError) Error() string {
	return e.ErrorString
}

// Message on the base error structure just delegates to Error()
func (e BaseError) Message() string {
	return e.Error()
}

// Propagate returns true for all BaseError implementations, as they're
// considered somewhat unknown and therefore more likely to be severe problems
func (e BaseError) Propagate() bool {
	return true
}

// Warning defaults to false for the same reason Propagate defaults to true.
// This can be true in other implementations to indicate users need to be given
// a warning they can choose to ignore.  (e.g., issue being semi-new, but not
// new enough to block queueing)
func (e BaseError) Warning() bool {
	return false
}

// New creates a new Error and returns it
func New(err string) Error {
	return &BaseError{err}
}

// Errorf stands in for fmt.Errorf as a simpler way to generate an Error
func Errorf(format string, args ...any) Error {
	return New(fmt.Sprintf(format, args...))
}

// List simplifies places where we need multiple errors
type List struct {
	Source []Error
}

// Major returns only non-warning errors - those which need to halt a process
func (l *List) Major() *List {
	var errs = new(List)
	for _, e := range l.Source {
		if !e.Warning() {
			errs.Append(e)
		}
	}
	return errs
}

// Minor returns all errors flagged as warnings - those which users can ignore
func (l *List) Minor() *List {
	var errs = new(List)
	for _, e := range l.Source {
		if e.Warning() {
			errs.Append(e)
		}
	}
	return errs
}

// Append adds e to the end of the Error list
func (l *List) Append(e Error) {
	l.Source = append(l.Source, e)
}

// All returns a raw list of errors - mainly for readability:
// foo.Errors.All() vs. foo.Errors.Source
func (l *List) All() []Error {
	return l.Source
}

// Clear empties the error list
func (l *List) Clear() {
	l.Source = nil
}

// Len returns the number of elements in the list
func (l *List) Len() int {
	return len(l.Source)
}
