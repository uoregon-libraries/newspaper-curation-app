package batchxml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html/template"
	"io"
	"os"
	"sort"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// Transformer takes a models.Batch and a list of models.Issues, and generates the XML
// to a given file
type Transformer struct {
	tmpl    *template.Template
	outFile string
	d       *data
	err     error
}

type data struct {
	*models.Batch
	Issues []*models.Issue
}

// New returns a batch XML Transformer
//
// We need a batch as well as all issues in order to avoid DB lookups, reduce
// unknowns, and allow for unsaved / faked data
func New(templatePath string, outputFileName string, batch *models.Batch, issues []*models.Issue) *Transformer {
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

	// Make sure the issues are sorted in a way that makes them easier to test
	// and debug
	sort.Slice(t.d.Issues, func(i, j int) bool {
		return t.d.Issues[i].Key() < t.d.Issues[j].Key()
	})

	var buf = new(bytes.Buffer)
	var err = t.tmpl.Execute(buf, t.d)
	if err != nil {
		return fmt.Errorf("unable to execute batch XML template: %w", err)
	}

	// Write to temp file, then copy if we're successful
	var f *os.File
	f, err = fileutil.TempFile("", "", "")
	if err != nil {
		return fmt.Errorf("unable to create batch XML temp output file: %w", err)
	}
	defer f.Close()
	defer os.Remove(f.Name())

	_, err = f.Write([]byte(xml.Header))
	if err == nil {
		_, err = io.Copy(f, buf)
	}
	if err != nil {
		return fmt.Errorf("unable to write to batch XML temp output file: %w", err)
	}

	err = fileutil.CopyFile(f.Name(), t.outFile)
	if err != nil {
		os.Remove(t.outFile)
		return fmt.Errorf("unable to write to batch XML temp output file: %w", err)
	}

	return nil
}
