package chronam

import (
	"encoding/xml"
	"fmt"
	"os"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

// BatchXML is used to deserialize batch.xml files to get at their issues list
type BatchXML struct {
	XMLName xml.Name        `xml:"batch"`
	Issues  []BatchIssueXML `xml:"issue"`
}

// BatchIssueXML describes each <issue> element in the batch XML
type BatchIssueXML struct {
	EditionOrder string `xml:"editionOrder,attr"`
	Date         string `xml:"issueDate,attr"`
	LCCN         string `xml:"lccn,attr"`
	Content      string `xml:",innerxml"`
}

// ParseBatchXML opens the given batch.xml file for a given batch path and
// returns the decoded XML structure.  This is intended primarily for
// retrieving the list of issues in a batch.
func ParseBatchXML(xmlFile string) (*BatchXML, error) {
	if !fileutil.IsFile(xmlFile) {
		return nil, fmt.Errorf("%q is not a file", xmlFile)
	}

	var contents, err = os.ReadFile(xmlFile)
	if err != nil {
		return nil, fmt.Errorf("batch XML file (%q) can't be read: %w", xmlFile, err)
	}

	var bx BatchXML
	err = xml.Unmarshal(contents, &bx)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal batch XML %q: %w", xmlFile, err)
	}

	return &bx, nil
}
