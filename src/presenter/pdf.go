package presenter

import (
	"sftp"
)

// PDF wraps an sftp PDF for presentation logic
type PDF struct {
	*sftp.PDF
	Issue *Issue
}

// DecoratePDF returns a wrapped sftp PDF for the given issue
func DecoratePDF(i *Issue, pdf *sftp.PDF) *PDF {
	return &PDF{pdf, i}
}
