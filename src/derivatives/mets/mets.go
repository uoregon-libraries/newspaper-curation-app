package mets

import (
	"db"
	"fmt"
)

// Transformer takes an issue and generates METS XML to a given file
type Transformer struct {
	outFile string
	issue   *db.Issue
	title   *db.Title
}

// New returns a METS Transformer
//
// We need an issue as well as a title in order to avoid DB lookups, reduce
// unknowns, and allow for unsaved / faked data
func New(templatePath string, outputFileName string, issue *db.Issue, title *db.Title) *Transformer {
	var t = &Transformer{outputFileName, issue, title}
	return t
}

// Transform generates the METS XML
func (t *Transformer) Transform() error {
	return fmt.Errorf("not implemented")
}
