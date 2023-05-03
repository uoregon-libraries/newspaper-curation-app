package alto

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"html/template"
	"unicode"
	"unicode/utf8"
)

// templateVars is used to inject data into the ALTO XML template
type templateVars struct {
	PDFFilename string
	PageWidth   int
	PageHeight  int
	ImageNumber int
	Flows       []Flow
	LangCode3   string
}

// scale uses ScaleFactor to multiply various x/y/width/height values so the
// ALTO data is properly set up for the actual image size
func (t *Transformer) scale(val float64) float64 {
	return val * t.ScaleFactor
}

func (t *Transformer) transform() {
	// Safety first!
	if t.err != nil {
		return
	}

	t.Logger.Infof("Converting pdftotext HTML to ALTO XML")

	// Pre-strip any super-busted runes (control characters and invalid runes)
	var cleaned []byte
	var offset int
	for offset < len(t.html) {
		var r, w = utf8.DecodeRune(t.html[offset:])
		offset += w
		if r == utf8.RuneError || unicode.IsControl(r) {
			continue
		}
		cleaned = utf8.AppendRune(cleaned, r)
	}

	// Parse XML to get at page attributes
	var source Doc
	var err = xml.Unmarshal(cleaned, &source)
	if err != nil {
		t.err = fmt.Errorf("invalid html to unmarshal into XML: %w", err)
		return
	}

	// Fix all "word" elements to avoid non-printable runes
	var html = source.Clean()

	// Set up template vars
	var blockNum int
	var funcs = template.FuncMap{
		"NextBlockNumber": func() int {
			blockNum++
			return blockNum
		},
		"MakeCoordAttrs": func(r Rect) template.HTMLAttr {
			var top = t.scale(r.YMin)
			var left = t.scale(r.XMin)
			var height = t.scale(r.YMax) - top
			var width = t.scale(r.XMax) - left

			var outfmt = `HEIGHT="%0.1f" WIDTH="%0.1f" HPOS="%0.1f" VPOS="%0.1f"`
			return template.HTMLAttr(fmt.Sprintf(outfmt, height, width, left, top))
		},
	}
	var altoTemplate = template.Must(template.New("alto").Funcs(funcs).Parse(altoTemplateString))
	var tvar = &templateVars{
		PDFFilename: t.PDFFilename,
		PageWidth:   int(t.scale(html.Page.Width)),
		PageHeight:  int(t.scale(html.Page.Height)),
		ImageNumber: t.ImageNumber,
		Flows:       html.Page.Flows,
		LangCode3:   t.LangCode3,
	}

	var buf = &bytes.Buffer{}
	err = altoTemplate.Execute(buf, tvar)
	if err != nil {
		t.err = fmt.Errorf("unable to run ALTO template: %w", err)
		return
	}

	t.xml = buf.Bytes()
}
