// pdftotext.go contains the data structures necessary for reading the "html"
// output from pdftotext

package alto

import (
	"unicode"
)

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

// Clean returns a copy of d with all non-printable unicode removed from all
// "word" elements, and any empty elements removed entirely
func (d Doc) Clean() Doc {
	var cleaned = d
	cleaned.Page = d.Page.clean()

	return cleaned
}

// Page holds the outer <page> wrapper around all the <flow> elements
type Page struct {
	Flows  []Flow  `xml:"flow"`
	Width  float64 `xml:"width,attr"`
	Height float64 `xml:"height,attr"`
}

func (p Page) clean() Page {
	var cleaned = p
	cleaned.Flows = nil

	for _, flow := range p.Flows {
		var f = flow.clean()
		if len(f.Blocks) > 0 {
			cleaned.Flows = append(cleaned.Flows, f)
		}
	}

	return cleaned
}

// Flow is just a container of blocks, theoretically grouped in a meaningful
// way (though this isn't always the case with PDFs that we see)
type Flow struct {
	Blocks []Block `xml:"block"`
}

func (f Flow) clean() Flow {
	var cleaned = f
	cleaned.Blocks = nil

	for _, block := range f.Blocks {
		var b = block.clean()
		if len(b.Lines) > 0 {
			cleaned.Blocks = append(cleaned.Blocks, b)
		}
	}

	return cleaned
}

// A Block contains lines and a rectangle around them
type Block struct {
	Rect
	Lines []Line `xml:"line"`
}

func (b Block) clean() Block {
	var cleaned = b
	cleaned.Lines = nil

	for _, line := range b.Lines {
		var l = line.clean()
		if len(l.Words) > 0 {
			cleaned.Lines = append(cleaned.Lines, l)
		}
	}

	return cleaned
}

// A Line contains the individual word elements, and a rectangle
type Line struct {
	Rect
	Words []Word `xml:"word"`
}

func (l Line) clean() Line {
	var cleaned = l
	cleaned.Words = nil

	for _, word := range l.Words {
		var w = word.clean()
		if w.Text != "" {
			cleaned.Words = append(cleaned.Words, w)
		}
	}

	return cleaned
}

// A Word is the most granular element we get, containing a rectangle around
// the text and the text itself
type Word struct {
	Rect
	Text string `xml:",chardata"`
}

func (w Word) clean() Word {
	var cleaned = w
	cleaned.Text = ""
	for _, r := range []rune(w.Text) {
		if unicode.In(r, unicode.Cc, unicode.Co, unicode.Cs) {
			continue
		}
		cleaned.Text = cleaned.Text + string(r)
	}

	return cleaned
}
