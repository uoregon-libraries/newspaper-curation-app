package titlehandler

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"regexp"

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
	var marcURL = "https://chroniclingamerica.loc.gov/lccn/" + t.LCCN + "/marc.xml"
	var resp, err = http.Get(marcURL)
	// An error from the Get call is not a deal-breaker, though we do want to
	// report it
	if err != nil {
		logger.Errorf("Unable to pull MARC XML from %q: %s", marcURL, err)
		return
	}
	defer resp.Body.Close()

	var data []byte
	data, err = ioutil.ReadAll(resp.Body)
	// An error reading the response is also not a deal-breaker, but a bit weirder
	if err != nil {
		logger.Errorf("Unable to read response body while pulling MARC XML from %q: %s", marcURL, err)
		return
	}

	var m marc
	xml.Unmarshal(data, &m)
	for _, df := range m.Datafields {
		if df.Tag == "245" {
			for _, sf := range df.Subfields {
				if sf.Code == "a" {
					t.MarcTitle = sf.Data
				}
			}
		}

		if df.Tag == "260" || df.Tag == "264" {
			for _, sf := range df.Subfields {
				if sf.Code == "a" {
					t.MarcLocation = marcStripLocRE.ReplaceAllString(sf.Data, "")
				}
			}
		}
	}

	if t.MarcTitle != "" && t.MarcLocation != "" {
		t.ValidLCCN = true
	}

	// Hopefully this saves, but if not we're not losing irreplacable data, so we just log the error and move on
	err = t.Save()
	if err != nil {
		logger.Errorf("Unable to save title (id %d) after MARC data pull: %s", t.ID, err)
	}
}
