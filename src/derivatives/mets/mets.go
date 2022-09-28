package mets

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// TimeFormat is the standard format used in our METS header - it's basically
// RFC3339 without a timezone
const TimeFormat = "2006-01-02T15:04:05"

// Transformer takes an issue and generates METS XML to a given file
type Transformer struct {
	tmpl    *template.Template
	outFile string
	d       *data
	err     error
}

type data struct {
	Issue      *models.Issue
	Pages      []*Page
	Title      *models.Title
	NowRFC3339 string
}

// New returns a METS Transformer
//
// We need an issue as well as a title in order to avoid DB lookups, reduce
// unknowns, and allow for unsaved / faked data
func New(templatePath string, outputFileName string, issue *models.Issue, title *models.Title, createDate time.Time) *Transformer {
	var tmpl = template.New("metsxml")
	var pgs, err = pages(issue)
	var t = &Transformer{
		tmpl:    tmpl,
		outFile: outputFileName,
		d: &data{
			Issue:      issue,
			Pages:      pgs,
			Title:      title,
			NowRFC3339: createDate.Format(TimeFormat),
		},
	}
	if err != nil {
		t.err = fmt.Errorf("unable to aggregate issue's pages: %w", err)
		return t
	}

	t.tmpl, t.err = tmpl.ParseFiles(templatePath)
	return t
}

// Transform generates the METS XML
func (t *Transformer) Transform() error {
	if t.err != nil {
		return t.err
	}

	var buf = new(bytes.Buffer)
	var err = t.tmpl.Execute(buf, t.d)
	if err != nil {
		return fmt.Errorf("unable to execute METS template: %w", err)
	}

	// Write to temp file, then copy if we're successful
	var f = fileutil.NewSafeFile(t.outFile)
	if f.Err != nil {
		f.Cancel()
		return fmt.Errorf("unable to create METS temp output file: %w", f.Err)
	}

	f.Write([]byte(xml.Header))
	io.Copy(f, buf)
	f.Close()
	if f.Err != nil {
		return fmt.Errorf("unable to write to METS temp output file: %w", f.Err)
	}

	return nil
}
