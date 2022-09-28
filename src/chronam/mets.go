package chronam

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"

	"github.com/uoregon-libraries/newspaper-curation-app/src/mods"
)

// A METSIssue stores deserialized issue XML, but at this time it *cannot* be
// reserialized due to bugs in how Go handles namespaces.  This is primarily
// for converting live issues into data structures which can then be serialized
// out again to verify the correctness of our custom METS output.
type METSIssue struct {
	XMLName xml.Name              `xml:"mets"`
	Label   string                `xml:"LABEL,attr"`
	Header  METSHeader            `xml:"metsHdr"`
	DMDSecs []DescriptiveMetadata `xml:"dmdSec"`
}

// METSHeader just gives us the XML file's creation date
type METSHeader struct {
	CreateDate string `xml:"CREATEDATE,attr"`
}

// DescriptiveMetadata holds the <dmdSec> stuff
type DescriptiveMetadata struct {
	ID   string    `xml:"ID,attr"`
	Data mods.Data `xml:"mdWrap>xmlData>mods"`
}

// ParseMETSIssueXML reads the given file to extract the relevant METSIssue data
func ParseMETSIssueXML(filename string) (*METSIssue, error) {
	var contents, err = ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var mets METSIssue
	err = xml.Unmarshal(contents, &mets)
	if err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &mets, nil
}
