package uploads

import (
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/pdf"
	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// File wraps a schema file to add upload-specific validations
type File struct {
	*schema.File
}

// ValidateDPI adds errors to the file if its embedded images' DPIs are not
// within 15% of the expected value.  This does nothing if the file isn't a pdf.
func (f *File) ValidateDPI(expected int) {
	if strings.ToUpper(filepath.Ext(f.Name)) != ".PDF" {
		return
	}

	var maxDPI = float64(expected) * 1.15
	var minDPI = float64(expected) * 0.85

	var dpis = pdf.ImageDPIs(f.Location)
	if len(dpis) == 0 {
		f.AddError(apperr.Errorf("contains no images or is invalid PDF"))
	}

	for _, dpi := range dpis {
		if dpi.X > maxDPI || dpi.Y > maxDPI || dpi.X < minDPI || dpi.Y < minDPI {
			f.AddError(apperr.Errorf("has an image with a bad DPI (%g x %g; expected DPI %d)", dpi.X, dpi.Y, expected))
		}
	}
}
