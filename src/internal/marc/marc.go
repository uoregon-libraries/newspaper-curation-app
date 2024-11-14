// Package marc has *extremely* rudimentary MARC XML processing for getting at
// a title's name, LCCN, and language code
package marc

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
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
	err = xml.Unmarshal(data, &mx)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling xml: %w", err)
	}

	var marc = &MARC{}

	for _, df := range mx.Datafields {
		if df.Tag == "010" {
			for _, sf := range df.Subfields {
				if sf.Code == "a" {
					marc.LCCN = sf.Data
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
