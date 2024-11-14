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
	LCCN     string
	Title    string
	Location string
	Language string
}

// ParseXML returns a new MARC instance from the XML in the given [io.Reader]
func ParseXML(r io.Reader) (*MARC, error) {
	var data, err = io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading MARC xml: %w", err)
	}

	var mx marcXML
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
		err = xml.Unmarshal(data2, &mx)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling <record>: %w", err)
		}

	case "record":
		err = xml.Unmarshal(data, &mx)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling <record>: %w", err)
		}

	default:
		return nil, fmt.Errorf(`unmarshaling xml: root node should be "collection" or "record" (got %q)`, root.XMLName.Local)
	}

	var marc = &MARC{}

	for _, df := range mx.Datafields {
		if df.Tag == "010" {
			for _, sf := range df.Subfields {
				if sf.Code == "a" {
					marc.LCCN = strings.Replace(sf.Data, " ", "", -1)
				}
			}
		}

		if df.Tag == "245" {
			for _, sf := range df.Subfields {
				if sf.Code == "a" {
					marc.Title = sf.Data
				}
			}
		}

		if df.Tag == "260" || df.Tag == "264" {
			for _, sf := range df.Subfields {
				if sf.Code == "a" {
					marc.Location = marcStripLocRE.ReplaceAllString(sf.Data, "")
				}
			}
		}
	}
	for _, cf := range mx.Controlfields {
		if cf.Tag == "008" {
			runes := []rune(cf.Data)
			marc.Language = string(runes[35:38])
		}
	}

	return marc, nil
}
