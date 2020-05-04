package mets

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// Page represents the data we need for all of an issue's pages: page number
// (1, 2, 3, etc.), prefix (0005, 0006, etc.) and label ("PAGE ONE", etc.)
type Page struct {
	Number int
	Prefix string
	Label  string
}

// HasLabel is true as long as the page label has a non-zero value
func (p *Page) HasLabel() bool {
	return p.Label != "0" && p.Label != ""
}

func stripExt(path string) string {
	for i := len(path) - 1; i >= 0 && !os.IsPathSeparator(path[i]); i-- {
		if path[i] == '.' {
			return path[:i]
		}
	}
	return path
}

// pages returns an ordered list of Page data
func pages(i *models.Issue) (pages []*Page, err error) {
	var labels = i.PageLabels

	var si *schema.Issue
	si, err = i.SchemaIssue()
	if err != nil {
		return nil, err
	}
	si.FindFiles()

	var pdfs []string
	for _, file := range si.Files {
		if filepath.Ext(file.Name) == ".pdf" {
			pdfs = append(pdfs, file.Name)
		}
	}

	if len(labels) != len(pdfs) {
		return nil, fmt.Errorf("%d labels found, but %d pdf files", len(labels), len(pdfs))
	}

	for i, pdf := range pdfs {
		var page = &Page{
			Number: i + 1,
			Prefix: stripExt(pdf),
			Label:  labels[i],
		}
		logger.Infof("Found page %#v", page)
		pages = append(pages, page)
	}

	return pages, err
}
