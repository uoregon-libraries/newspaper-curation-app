// Package mods holds data structures to simplify unmarshaling the Issue XML
// which holds lots of MODS structures
package mods

// Data holds the meat of the issue XML
type Data struct {
	RelatedItems []RelItem    `xml:"relatedItem"`
	OriginInfos  []OriginInfo `xml:"originInfo"`
	Parts        []Part       `xml:"part"`
	Rights       string       `xml:"accessCondition"`
}

// RelItem holds high-level issue metadata when in the issue metadata
// section; it's ignored for the page metadata
type RelItem struct {
	Type  string `xml:"type,attr"`
	IDs   []ID   `xml:"identifier"`
	Parts []Part `xml:"part"`
}

// OriginInfo holds issue date and date-as-labeled
type OriginInfo struct {
	Dates []DateIssued `xml:"dateIssued"`
}

// Part seems to hold a lot of very generic information that can only be
// made sense of in context
type Part struct {
	Details []Detail `xml:"detail"`
	Extents []Extent `xml:"extent"`
}

// ID represents an id label and an id type; for our uses this is primarily
// just the issue's LCCN
type ID struct {
	Label string `xml:",chardata"`
	Type  string `xml:"type,attr"`
}

// DateIssued holds dates and qualifiers like "questionable"
type DateIssued struct {
	Date      string `xml:",chardata"`
	Qualifier string `xml:"qualifier,attr"`
}

// Detail holds numbers (which aren't really numbers) and captions.  The type
// attribute tells us what the number and/or caption mean.
type Detail struct {
	Type    string `xml:"type,attr"`
	Number  string `xml:"number"`
	Caption string `xml:"caption"`
}

// Extent holds... a range maybe?  I'm too lazy to look up the mods spec.
type Extent struct {
	Unit  string `xml:"unit,attr"`
	Start string `xml:"start"`
}
