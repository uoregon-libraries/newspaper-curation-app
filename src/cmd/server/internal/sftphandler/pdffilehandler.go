package sftphandler

import (
	"cmd/server/internal/responder"

	"fmt"
	"io"

	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
)

// PDFFileHandler attempts to find and display a PDF file to the browser
func PDFFileHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var issue = findIssue(r)
	var fileslug = mux.Vars(req)["filename"]
	var pdf = issue.PDFLookup[fileslug]
	if pdf == nil {
		r.Vars.Alert = fmt.Sprintf("Invalid request")
		r.Render(responder.Empty)
		return
	}

	var path = pdf.Location
	if strings.ToUpper(filepath.Ext(path)) != ".PDF" {
		r.Vars.Alert = fmt.Sprintf("%#v is not a valid PDF file and cannot be viewed", path)
		r.Render(responder.Empty)
		return
	}

	if !fileutil.IsFile(path) {
		r.Vars.Alert = fmt.Sprintf("Unable to locate %#v!", path)
		r.Render(responder.Empty)
		return
	}

	var f, err = os.Open(path)
	if err != nil {
		logger.Errorf("Unable to read %#v", path)
		r.Vars.Alert = fmt.Sprintf("Unable to read %#v!", path)
		r.Render(responder.Empty)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/pdf")
	io.Copy(w, f)
}
