package uploads

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/uoregon-libraries/gopkg/pdf"
	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// File wraps a schema file to add upload-specific validations
type File struct {
	*schema.File
}

// Overridable pdf.Images function for testing
var _dpifunc = func(loc string) ([]*pdf.Image, error) {
	return pdf.ImageInfo(loc)
}

// ValidateDPI adds errors to the file if its embedded images' DPIs are not
// within 15% of the expected value.  This does nothing if the file isn't a pdf.
func (f *File) ValidateDPI(expected int) {
	if strings.ToUpper(filepath.Ext(f.Name)) != ".PDF" {
		return
	}

	var maxDPI = float64(expected) * 1.15
	var minDPI = float64(expected) * 0.85

	var images, err = _dpifunc(f.Location)
	if err != nil {
		f.AddError(apperr.Errorf("unable to get image info: %s", err))
		return
	}

	if len(images) == 0 {
		f.AddError(apperr.Errorf("contains no images or is not a valid PDF"))
		return
	}

	// Abbyy is currently giving us PDF with tons of embedded images, many of
	// which are rather irrelevant, but it's impossible to tell exactly which
	// matter and which don't from a program.  So for now, all embedded images,
	// unless they're absurdly small, need to have our expected DPI.
	var width, height int
	var xdpi, ydpi float64
	var invalidImage bool
	for _, image := range images {
		width, _ = strconv.Atoi(image.Width)
		height, _ = strconv.Atoi(image.Height)
		if width*height < 1000 {
			continue
		}

		xdpi, _ = strconv.ParseFloat(image.XPPI, 64)
		ydpi, _ = strconv.ParseFloat(image.YPPI, 64)
		if xdpi < minDPI || xdpi > maxDPI || ydpi < minDPI || ydpi > maxDPI {
			invalidImage = true
			break
		}
	}

	if invalidImage {
		f.AddError(apperr.Errorf("contains one or more invalid images"))
	}
}
