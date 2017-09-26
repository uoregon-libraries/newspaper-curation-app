// pdftotext.go contains the data structures necessary for reading the "html"
// output from pdftotext

package alto

// Rect is a common structure embedded in most pdftotext elements
type Rect struct {
	XMin float64 `xml:"xMin,attr"`
	YMin float64 `xml:"yMin,attr"`
	XMax float64 `xml:"xMax,attr"`
	YMax float64 `xml:"yMax,attr"`
}

// Width returns the X difference
func (r Rect) Width() float64 {
	return r.XMax - r.XMin
}

// Height returns the Y difference
func (r Rect) Height() float64 {
	return r.YMax - r.YMin
}

// Doc is the outermost element in the pdftotext html; it should contain
// exactly one page in all cases for us
type Doc struct {
	Page Page `xml:"page"`
}

// Page holds the outer <page> wrapper around all the <flow> elements
type Page struct {
	Flows  []Flow  `xml:"flow"`
	Width  float64 `xml:"width,attr"`
	Height float64 `xml:"height,attr"`
}

// Flow is just a container of blocks, theoretically grouped in a meaningful
// way (though this isn't always the case with PDFs that we see)
type Flow struct {
	Blocks []Block `xml:"block"`
}

// A Block contains lines and a rectangle around them
type Block struct {
	Rect
	Lines []Line `xml:"line"`
}

// A Line contains the individual word elements, and a rectangle
type Line struct {
	Rect
	Words []Word `xml:"word"`
}

// A Word is the most granular element we get, containing a rectangle around
// the text and the text itself
type Word struct {
	Rect
	Text string `xml:",chardata"`
}
