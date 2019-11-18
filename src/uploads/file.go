package uploads

import (
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/logger"
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

	// Let's not spam logs with debug nonsense from the pdf package
	pdf.Logger = logger.Named("gopkg/pdf.ImageDPIs", logger.Warn)
	var dpis = pdf.ImageDPIs(f.Location)
	if len(dpis) == 0 {
		f.AddError(apperr.Errorf("contains no images or is invalid PDF"))
	}

	// We're willing to accept small DPIs sometimes, because we need to allow
	// for Abbyy's odd "mask"-like images it embeds.  This is extremely imperfect
	// and could easily let weird images through, but we can't really help that
	// without manual checks to isolate images which are relevant.  We try to
	// categorize the DPI oddities in a way that will make more sense - too-small
	// is annoying but doesn't actually give us problems, whereas too-big means
	// we may waste a lot of disk, so it's more critical.
	var tooSmall int
	for _, dpi := range dpis {
		if dpi.X > maxDPI || dpi.Y > maxDPI {
			f.AddError(apperr.Errorf("has an image with a bad DPI (%g x %g; expected DPI %d)", dpi.X, dpi.Y, expected))
		}
		if dpi.X < minDPI || dpi.Y < minDPI {
			tooSmall++
		}
	}
	if tooSmall >= len(dpis)/2 {
		f.AddError(apperr.Errorf("has too many images with a DPI below %d", expected))
	}
}
