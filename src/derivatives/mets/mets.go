package mets

import (
	"bytes"
	"db"
	"fmt"
	"html/template"
	"io"
	"os"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

// Transformer takes an issue and generates METS XML to a given file
type Transformer struct {
	tmpl    *template.Template
	outFile string
	issue   *db.Issue
	title   *db.Title
	err     error
}

type data struct {
	*db.Issue
	*db.Title
	NowRFC3339 string
}

// New returns a METS Transformer
//
// We need an issue as well as a title in order to avoid DB lookups, reduce
// unknowns, and allow for unsaved / faked data
func New(templatePath string, outputFileName string, issue *db.Issue, title *db.Title) *Transformer {
	var tmpl = template.New("metsxml")
	tmpl.Funcs(template.FuncMap{"incr": func(i int) int { return i + 1 }})
	var t = &Transformer{tmpl, outputFileName, issue, title, nil}
	t.tmpl, t.err = tmpl.ParseFiles(templatePath)
	return t
}

// Transform generates the METS XML
func (t *Transformer) Transform() error {
	if t.err != nil {
		return t.err
	}

	var buf = new(bytes.Buffer)
	var err = t.tmpl.Execute(buf, data{t.issue, t.title, time.Now().Format(time.RFC3339)})
	if err != nil {
		return fmt.Errorf("unable to execute METS template: %s", err)
	}

	// Write to temp file, then copy if we're successful
	var f *os.File
	f, err = fileutil.TempFile("", "", "")
	if err != nil {
		return fmt.Errorf("unable to create METS temp output file: %s", err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	_, err = io.Copy(f, buf)
	if err != nil {
		return fmt.Errorf("unable to write to METS temp output file: %s", err)
	}

	err = fileutil.CopyFile(f.Name(), t.outFile)
	if err != nil {
		os.Remove(t.outFile)
		return fmt.Errorf("unable to write to METS temp output file: %s", err)
	}

	return nil
}
