package titlehandler

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/marc"
)

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
		if i >= last {
			logger.Errorf(msg, marcLocs[i], err)
			return
		}
		logger.Warnf(msg+" -- trying next location", marcLocs[i], err)
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
	if err != nil {
		return fmt.Errorf("preparing MARC XML reader: %w", err)
	}
	defer reader.Close()

	var m *marc.MARC
	m, err = marc.ParseXML(reader)
	t.MARCTitle = m.Title()
	t.MARCLocation = m.Location()
	t.LangCode3 = m.Language()
	if t.MARCTitle == "" || t.MARCLocation == "" {
		return fmt.Errorf("parsing MARC XML: title and location must not be blank")
	}

	t.ValidLCCN = true

	err = t.Save()
	if err != nil {
		return fmt.Errorf("saving title (id %d) after MARC XML read: %w", t.ID, err)
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
