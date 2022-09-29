package titlehandler

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
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

type marc struct {
	Datafields    []datafield    `xml:"datafield"`
	Controlfields []controlfield `xml:"controlfield"`
}

// pullMARCForTitle pulls the MARC record from the library of congress and sets the
// title's data if successful
func pullMARCForTitle(t *Title) {
	t.ValidLCCN = false
	t.MARCTitle = ""
	t.MARCLocation = ""

	var marcLocs = []string{
		strings.Replace(conf.MARCLocation1, "{{lccn}}", t.LCCN, -1),
		strings.Replace(conf.MARCLocation2, "{{lccn}}", t.LCCN, -1),
	}

	var last = len(marcLocs) - 1
	for i := 0; i <= last; i++ {
		var err = lookupMARC(t, marcLocs[i])
		if err == nil {
			return
		}
		var msg = "Unable to pull MARC XML from %q: %s"
		if i < last {
			logger.Warnf(msg+" -- trying next location", marcLocs[i], err)
		} else {
			logger.Errorf(msg, marcLocs[i], err)
			return
		}
	}
}

func lookupMARC(t *Title, marcLoc string) error {
	logger.Infof("Looking up MARC for %q in configured location %q", t.LCCN, marcLoc)

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
		return err
	}
	defer reader.Close()

	var data []byte
	data, err = ioutil.ReadAll(reader)
	// An error reading the response is also not a deal-breaker, but a bit weirder
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
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
	for _, cf := range m.Controlfields {
		if cf.Tag == "008" {
			runes := []rune(cf.Data)
			t.LangCode3 = string(runes[35:38])
		}
	}
	if t.MARCTitle != "" && t.MARCLocation != "" {
		t.ValidLCCN = true
	} else {
		return fmt.Errorf("invalid xml response: title and location must not be blank")
	}

	// Hopefully this saves, but if not we're not losing irreplacable data, so we just log the error and move on
	err = t.Save()
	if err != nil {
		return fmt.Errorf("unable to save title (id %d) after MARC data pull: %w", t.ID, err)
	}

	return nil
}

func getMarcHTTP(uri string) (io.ReadCloser, error) {
	var resp, err = http.Get(uri)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func getMarcLocal(loc string) (io.ReadCloser, error) {
	return os.Open(loc)
}
