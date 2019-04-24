package batchxml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"os"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
)

// Transformer takes a db.Batch and a list of db.Issues, and generates the XML
// to a given file
type Transformer struct {
	tmpl    *template.Template
	outFile string
	d       *data
	err     error
}

type data struct {
	*db.Batch
	Issues []*db.Issue
}

// New returns a batch XML Transformer
//
// We need a batch as well as all issues in order to avoid DB lookups, reduce
// unknowns, and allow for unsaved / faked data
func New(templatePath string, outputFileName string, batch *db.Batch, issues []*db.Issue) *Transformer {
	var tmpl = template.New("batch")
	tmpl.Funcs(
		template.FuncMap{"incr": func(i int) int { return i + 1 }},
	)
	var t = &Transformer{tmpl, outputFileName, &data{batch, issues}, nil}
	t.tmpl, t.err = tmpl.ParseFiles(templatePath)
	return t
}

// Transform generates the batch XML
func (t *Transformer) Transform() error {
	if t.err != nil {
		return t.err
	}

	var buf = new(bytes.Buffer)
	var err = t.tmpl.Execute(buf, t.d)
	if err != nil {
		return fmt.Errorf("unable to execute batch XML template: %s", err)
	}

	// Write to temp file, then copy if we're successful
	var f *os.File
	f, err = fileutil.TempFile("", "", "")
	if err != nil {
		return fmt.Errorf("unable to create batch XML temp output file: %s", err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	f.Write([]byte(xml.Header))
	_, err = io.Copy(f, buf)
	if err != nil {
		return fmt.Errorf("unable to write to batch XML temp output file: %s", err)
	}

	err = fileutil.CopyFile(f.Name(), t.outFile)
	if err != nil {
		os.Remove(t.outFile)
		return fmt.Errorf("unable to write to batch XML temp output file: %s", err)
	}

	return nil
}
