package uploadedissuehandler

import (
	"cmd/server/internal/responder"

	"fmt"
	"io"

	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
)

// PDFFileHandler attempts to find and display a PDF file to the browser
func PDFFileHandler(w http.ResponseWriter, req *http.Request) {
	var r = getResponder(w, req)
	if r.err != nil {
		return
	}

	var fileslug = r.vars["filename"]
	var pdf = r.issue.PDFLookup[fileslug]
	if pdf == nil {
		r.Error(http.StatusBadRequest, "")
		return
	}

	var path = pdf.Location
	if strings.ToUpper(filepath.Ext(path)) != ".PDF" {
		r.Vars.Alert = fmt.Sprintf("%q is not a valid PDF file and cannot be viewed", path)
		r.Render(responder.Empty)
		return
	}

	if !fileutil.IsFile(path) {
		r.Error(http.StatusNotFound, fmt.Sprintf("Unable to locate %q!", path))
		return
	}

	var f, err = os.Open(path)
	if err != nil {
		logger.Errorf("Unable to read %q", path)
		r.Error(http.StatusInternalServerError, fmt.Sprintf("Unable to read %q!", path))
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/pdf")
	io.Copy(w, f)
}
