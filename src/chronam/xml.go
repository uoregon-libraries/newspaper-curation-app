package chronam

import (
	"encoding/xml"

	"fmt"
	"io/ioutil"
	"path/filepath"

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

// ParseBatchXML finds the batch.xml file for a given batch path and returns
// the decoded XML structure.  This is intended primarily for retrieving the
// list of issues in a batch.
func ParseBatchXML(batchDir string) (*BatchXML, error) {
	var xmlFile = filepath.Join(batchDir, "data", "batch.xml")
	if !fileutil.IsFile(xmlFile) {
		return nil, fmt.Errorf("batch directory %#v has no batch.xml", batchDir)
	}

	var contents, err = ioutil.ReadFile(xmlFile)
	if err != nil {
		return nil, fmt.Errorf("batch XML file (%#v) can't be read: %s", xmlFile, err)
	}

	var bx BatchXML
	err = xml.Unmarshal(contents, &bx)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal batch XML %#v: %s", xmlFile, err)
	}

	return &bx, nil
}
