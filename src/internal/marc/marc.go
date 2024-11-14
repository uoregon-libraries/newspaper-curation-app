// Package marc has *extremely* rudimentary MARC XML processing for getting at
// a title's name, LCCN, and language code
package marc

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/uoregon-libraries/gopkg/xmlnode"
)

var marcStripLocRE = regexp.MustCompile(`[ /:,]+$`)

type subfield struct {
	Code string `xml:"code,attr"`
	Data string `xml:",innerxml"`
}

type datafield struct {
	Subfields []subfield `xml:"subfield"`
	Ind1      string     `xml:"ind1,attr"`
	Ind2      string     `xml:"ind2,attr"`
	Tag       string     `xml:"tag,attr"`
}

type controlfield struct {
	Tag  string `xml:"tag,attr"`
	Data string `xml:",innerxml"`
}

type marcXML struct {
	Datafields    []datafield    `xml:"datafield"`
	Controlfields []controlfield `xml:"controlfield"`
}

// MARC holds the raw data parsed from an XML source
type MARC struct {
	raw    *marcXML
	fields map[string]string
}

func newMARC(raw *marcXML) *MARC {
	var m = &MARC{raw: raw, fields: make(map[string]string)}

	for _, cf := range raw.Controlfields {
		m.fields[cf.Tag] = cf.Data
	}
	for _, df := range raw.Datafields {
		for _, sf := range df.Subfields {
			m.fields[df.Tag+"$"+sf.Code] = sf.Data
		}
	}

	return m
}

// Get returns the value of the field with the given tag. Control fields, such
// as "008", have no code, and can be requested directly. Data fields have
// subfields, and must include a tag to indicate which subfield, e.g., tag
// "245" and code "a".
func (m *MARC) Get(tag, code string) string {
	if code == "" {
		return m.fields[tag]
	}
	return m.fields[tag+"$"+code]
}

// LCCN returns field 010 $a, stripped of all spaces
func (m *MARC) LCCN() string {
	return strings.Replace(m.Get("010", "a"), " ", "", -1)
}

// Title returns field 245 $a from MARC
func (m *MARC) Title() string {
	return strings.TrimSpace(m.Get("245", "a"))
}

// Location returns the value in field 260 $a or 264 $a, with special
// characters removed. Field 264 is given precedence.
func (m *MARC) Location() string {
	var location = m.Get("264", "a")
	if location == "" {
		location = m.Get("260", "a")
	}

	return marcStripLocRE.ReplaceAllString(location, "")
}

// Language returns the three-character language code from field 008
func (m *MARC) Language() string {
	var lang = []rune(m.Get("008", ""))
	if len(lang) < 38 {
		return ""
	}

	return string(lang[35:38])
}

// parse is our low-level XML parser that gets the raw data structure set up,
// but doesn't do any data processing / translating
func parse(r io.Reader) (*marcXML, error) {
	var data, err = io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading MARC xml: %w", err)
	}

	var mx = new(marcXML)
	var root = new(xmlnode.Node)
	err = xml.Unmarshal(data, root)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling xml into generic structure: %w", err)
	}
	switch root.XMLName.Local {
	case "collection":
		if len(root.Nodes) == 0 {
			return nil, fmt.Errorf("parsing generic xml: root node has no children")
		}
		if len(root.Nodes) > 1 {
			return nil, fmt.Errorf("parsing generic xml: root node has too many children")
		}
		var data2, err = xml.Marshal(root.Nodes[0])
		if err != nil {
			return nil, fmt.Errorf("parsing generic xml: internal error re-exporting <record>: %w", err)
		}
		err = xml.Unmarshal(data2, mx)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling <record>: %w", err)
		}

	case "record":
		err = xml.Unmarshal(data, mx)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling <record>: %w", err)
		}

	default:
		return nil, fmt.Errorf(`unmarshaling xml: root node should be "collection" or "record" (got %q)`, root.XMLName.Local)
	}

	return mx, nil
}

// ParseXML returns a new MARC instance from the XML in the given [io.Reader]
func ParseXML(r io.Reader) (*MARC, error) {
	var mx, err = parse(r)
	if err != nil {
		return nil, err
	}

	return newMARC(mx), nil
}
