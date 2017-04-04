package presenter

import (
	"fmt"
	"html/template"
	"path/filepath"
	"sftp"
	"web/webutil"
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

// Link returns the link to view/download the PDF if it is a *.pdf file,
// otherwise just its name is returned
func (pdf *PDF) Link() template.HTML {
	if filepath.Ext(pdf.Name) != ".pdf" {
		return template.HTML(pdf.Name)
	}

	var path = webutil.PDFPath(pdf.Issue.Publisher.Name, pdf.Issue.Name, pdf.Name)
	return template.HTML(fmt.Sprintf(`
		<a href="%s" target="_blank">%s</a>
		<span class="sr-only">(opens in a new tab)</span>`, path, pdf.Name))
}
