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

// List simplifies places where we need multiple errors
type List []Error

// New creates a new Error and returns it
func New(err string) Error {
	return &BaseError{err}
}

// Errorf stands in for fmt.Errorf as a simpler way to generate an Error
func Errorf(format string, args ...interface{}) Error {
	return New(fmt.Sprintf(format, args...))
}
