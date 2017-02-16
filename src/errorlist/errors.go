// Package errorlist helps manage lists of error objects
package errorlist

// Errors wraps an error list with some helper functions
type Errors struct {
	errors []error
}

func New() *Errors {
	return &Errors{}
}

// Append adds err to the error list
func (l *Errors) Append(err error) {
	l.errors = append(l.errors, err)
}

func (l *Errors) String() string {
	var out = ""
	for i, err := range l.errors {
		if i > 0 {
			out += "; "
		}
		out += err.Error()
	}

	return out
}

// Len returns the number of items in the error list
func (l *Errors) Len() int {
	return len(l.errors)
}
