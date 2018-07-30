package titlehandler

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/uoregon-libraries/gopkg/logger"
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

type marc struct {
	Datafields []datafield `xml:"datafield"`
}

// pullMARCForTitle pulls the MARC record from the library of congress and sets the
// title's data if successful
func pullMARCForTitle(t *Title) {
	t.ValidLCCN = false
	t.MARCTitle = ""
	t.MARCLocation = ""

	var marcLoc = strings.Replace(conf.MARCLocation, "{{lccn}}", t.LCCN, -1)
	var reader io.ReadCloser
	var err error

	if marcLoc[:4] == "http" {
		reader, err = getMarcHTTP(marcLoc)
	} else {
		reader, err = getMarcLocal(marcLoc)
	}

	// An error from the Get call is not a deal-breaker, though we do want to
	// report it
	if err != nil {
		logger.Errorf("Unable to pull MARC XML from %q: %s", marcLoc, err)
		return
	}
	defer reader.Close()

	var data []byte
	data, err = ioutil.ReadAll(reader)
	// An error reading the response is also not a deal-breaker, but a bit weirder
	if err != nil {
		logger.Errorf("Unable to read response body while pulling MARC XML from %q: %s", marcLoc, err)
		return
	}

	var m marc
	xml.Unmarshal(data, &m)
	for _, df := range m.Datafields {
		if df.Tag == "245" {
			for _, sf := range df.Subfields {
				if sf.Code == "a" {
					t.MARCTitle = sf.Data
				}
			}
		}

		if df.Tag == "260" || df.Tag == "264" {
			for _, sf := range df.Subfields {
				if sf.Code == "a" {
					t.MARCLocation = marcStripLocRE.ReplaceAllString(sf.Data, "")
				}
			}
		}
	}

	if t.MARCTitle != "" && t.MARCLocation != "" {
		t.ValidLCCN = true
	}

	// Hopefully this saves, but if not we're not losing irreplacable data, so we just log the error and move on
	err = t.Save()
	if err != nil {
		logger.Errorf("Unable to save title (id %d) after MARC data pull: %s", t.ID, err)
	}
}

func getMarcHTTP(uri string) (io.ReadCloser, error) {
	var resp, err = http.Get(uri)
	return resp.Body, err
}

func getMarcLocal(loc string) (io.ReadCloser, error) {
	return os.Open(loc)
}
