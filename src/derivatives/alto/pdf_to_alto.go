package alto

import (
	"bytes"
	"fileutil"
	"fmt"
	"io/ioutil"
	"logger"
	"os"
	"regexp"
	"shell"
)

// lowASCIIRegex strips all low-ASCII that isn't printable
var lowASCIIRegex = regexp.MustCompile(`[\x00-\x08\x0b\x0c\x0e-\x1f]`)

// Transformer holds onto various data needed to convert a PDF into
// ALTO-compatible XML, halting the process at the first error
type Transformer struct {
	PDFFilename        string
	ALTOOutputFilename string
	ScaleFactor        float64
	ImageNumber        int

	// Logger can be set up manually for customized logging, otherwise it just
	// gets set to the default logger
	Logger logger.Logger

	err  error
	html []byte
	xml  []byte
}

// New sets up a new transformer to convert a PDF to ALTO XML
func New(pdfFile, altoFile string, pdfDPI float64, imgNo int) *Transformer {
	return &Transformer{
		PDFFilename:        pdfFile,
		ALTOOutputFilename: altoFile,
		ScaleFactor:        pdfDPI / 72.0,
		ImageNumber:        imgNo,
		Logger:             logger.DefaultLogger,
	}
}

// Transform takes the PDF file and runs it through pdftotext, then strips
// extraneous data from the generated HTML file, and finally writes an
// ALTO-like XML file to ALTOOutputFilename.  If the return is anything but
// nil, the ALTO XML will not have been created.
func (t *Transformer) Transform() error {
	// File existence is not a failure; just means we don't regenerate the file
	if fileutil.Exists(t.ALTOOutputFilename) {
		t.Logger.Info("Not generating ALTO XML file %q; file already exists", t.ALTOOutputFilename)
		return nil
	}

	t.pdfToText()
	t.extractDoc()
	t.transform()
	t.writeALTOFile()

	return t.err
}

// pdfToText runs the pdftotext binary and stores the HTML generated
func (t *Transformer) pdfToText() {
	// Safety first!
	if t.err != nil {
		return
	}

	t.Logger.Info("Running pdftotext on %q", t.PDFFilename)

	var f, err = ioutil.TempFile("", "")
	if err != nil {
		t.err = fmt.Errorf("unable to create tempfile for HTML output: %s", err)
		return
	}
	var tmpfile = f.Name()
	f.Close()
	defer os.Remove(tmpfile)

	if !shell.Exec("pdftotext", t.PDFFilename, "-bbox-layout", tmpfile) {
		t.err = fmt.Errorf("unable to run pdftotext")
		return
	}

	f, err = os.Open(tmpfile)
	if err != nil {
		t.err = fmt.Errorf("error opening HTML file: %s", err)
		return
	}
	defer f.Close()

	t.html, err = ioutil.ReadAll(f)
	if err != nil {
		t.err = fmt.Errorf("error reading HTML file: %s", err)
	}
}

// extractDoc pulls the relevant HTML out of the file passed in, stripping
// unnecessary cruft from the pdftohtml html process and storing it
func (t *Transformer) extractDoc() {
	// Safety first!
	if t.err != nil {
		return
	}

	t.Logger.Info("Extracting XML")

	var start = bytes.Index(t.html, []byte("<doc>"))
	var end = bytes.Index(t.html, []byte("</doc>"))
	t.html = t.html[start : end+6]
	t.html = lowASCIIRegex.ReplaceAllLiteral(t.html, nil)
}

func (t *Transformer) writeALTOFile() {
	// Safety first!
	if t.err != nil {
		return
	}

	t.Logger.Info("Writing out ALTO XML to %q", t.ALTOOutputFilename)

	var f, err = os.Create(t.ALTOOutputFilename)
	if err != nil {
		t.err = fmt.Errorf("unable to create alto output file %q: %s", t.ALTOOutputFilename, err)
		return
	}

	_, err = f.Write(t.xml)
	if err != nil {
		t.err = fmt.Errorf("unable to write to alto output file %q: %s", t.ALTOOutputFilename, err)
		f.Close()
		os.Remove(t.ALTOOutputFilename)
		return
	}

	f.Close()
}
