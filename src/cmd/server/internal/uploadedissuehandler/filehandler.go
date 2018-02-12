package uploadedissuehandler

import (
	"cmd/server/internal/responder"
	"mime"

	"fmt"
	"io"

	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
)

// FileHandler attempts to find and display a file to the browser
func FileHandler(w http.ResponseWriter, req *http.Request) {
	var r = getResponder(w, req)
	if r.err != nil {
		r.Render(nil)
		return
	}

	var fileslug = r.vars["filename"]
	var file = r.issue.FileLookup[fileslug]
	if file == nil {
		r.Error(http.StatusBadRequest, "")
		return
	}

	var path = file.Location
	var ext = strings.ToUpper(filepath.Ext(path))
	if ext != ".PDF" && ext != ".TIF" && ext != ".TIFF" {
		r.Vars.Alert = fmt.Sprintf("%q is not a valid file type (PDF/TIFF only), and cannot be viewed", path)
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

	w.Header().Set("Content-Type", mime.TypeByExtension(ext))
	io.Copy(w, f)
}
