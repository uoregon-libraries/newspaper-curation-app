// Package jp2 converts a PDF or TIFF into a JP2.  The resulting JP2 is then
// verified as being readable to avoid catching encoding problems "too late".
package jp2

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/uoregon-libraries/gopkg/fileutil"
	ltype "github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

// RateFactor is our constant divisor / multiplier for rates.  We store
// testable rates as ints multiplied by this value so they're easily keyed in a
// hash, and divide by this value to get a float when running an actual encode
// operation.
const RateFactor = 8.0

// Transformer is the PDF/TIFF-to-JP2 structure.  Paths to various binaries
// have sane defaults, but can be overridden if necessary.  A custom
// logger.Logger can be used for logging, otherwise the default logger is used.
type Transformer struct {
	SourceFile string
	OutputJP2  string
	tmpPNG     string
	tmpPNGTest string
	tmpJP2     string

	OPJCompress    string
	OPJDecompress  string
	GhostScript    string
	GraphicsMagick string
	Quality        float64
	PDFResolution  int
	OverwriteJP2   bool // if true, doesn't skip files which already exist

	err    error
	Logger *ltype.Logger
}

// New creates a new PDF/TIFF-to-JP2 transformer with default values for the
// various binaries and use of the default logger
func New(source, output string, quality float64, resolution int, overwrite bool) *Transformer {
	return &Transformer{
		SourceFile:     source,
		OutputJP2:      output,
		OPJCompress:    "opj_compress",
		OPJDecompress:  "opj_decompress",
		GhostScript:    "gs",
		GraphicsMagick: "gm",
		Quality:        quality,
		PDFResolution:  resolution,
		OverwriteJP2:   overwrite,
		Logger:         logger.Logger,
	}
}

// getRate returns roughly the GraphicsMagick approach to convert quality to
// JP2 rate values
func (t *Transformer) getRate() float64 {
	var d = 115.0 - t.Quality
	var r1 = 100.0 / (d * d)
	return 1.0 / r1
}

// Transform runs the conversions necessary to get from source to PNG to JP2,
// and then verifies the JP2 can be read (or else attempts to build it again
// using a different quality)
func (t *Transformer) Transform() error {
	if fileutil.Exists(t.OutputJP2) {
		if t.OverwriteJP2 {
			t.Logger.Debugf("Removing existing JP2 file %q", t.OutputJP2)
			var err = os.Remove(t.OutputJP2)
			if err != nil {
				return fmt.Errorf("removing existing JP2 in Transform(): %w", err)
			}
		} else {
			t.Logger.Infof("Not generating JP2 file %q; file already exists", t.OutputJP2)
			return nil
		}
	}

	t.makePNG()
	t.makeJP2()
	t.moveTempJP2()

	t.Logger.Debugf("Removing tmpPNG %q", t.tmpPNG)
	os.Remove(t.tmpPNG)
	t.Logger.Debugf("Removing tmpJP2 %q", t.tmpJP2)
	os.Remove(t.tmpJP2)
	t.Logger.Debugf("Removing tmpPNGTest %q", t.tmpPNGTest)
	os.Remove(t.tmpPNGTest)

	return t.err
}

// makePNG converts the source file to a PNG based on the source file's type
func (t *Transformer) makePNG() {
	// Safety first!
	if t.err != nil {
		return
	}
	var err error

	t.Logger.Infof("Creating PNG from %q", t.SourceFile)

	t.tmpPNG, err = fileutil.TempNamedFile("", "", ".png")
	if err != nil {
		t.err = fmt.Errorf("unable to create temporary PNG: %w", err)
		return
	}

	var inType = filepath.Ext(t.SourceFile)
	var success bool
	switch inType {
	case ".pdf":
		success = t.makePNGFromPDF()
	case ".tiff", ".tif":
		success = t.makePNGFromTIFF()
	default:
		t.err = fmt.Errorf("cannot process %q (input file must be *.pdf or *.tiff)", t.SourceFile)
		return
	}

	if !success {
		t.err = fmt.Errorf("failed running PNG shell command")
		return
	}
}

// makeJP2 loops through various rates attempting to build and verify a JP2.
// This is a terrible hack to deal with the odd, rare PNG which won't convert
// to a readable JP2.  The problem occurs about 1% of the time, so we have to
// do this or else we can lose a huge percentage of our born-digital issues,
// since those can have dozens of pages each.
func (t *Transformer) makeJP2() {
	// Safety first!
	if t.err != nil {
		return
	}
	var err error

	t.Logger.Infof("Creating JP2 from PNG")

	// Create a temp file for holding our JP2.
	//
	// We cannot retrofit this to use fileutil.SafeFile because we have to shell
	// out to commands to write to the file.  An io.Writer can't be used, and
	// capturing things like errors on Write, Close, etc. isn't possible.
	t.tmpJP2, err = fileutil.TempNamedFile("", "", ".jp2")
	if err != nil {
		t.err = fmt.Errorf("unable to create test JP2: %w", err)
		return
	}

	// Create a stable test PNG for use in the JP2 decode verification(s)
	t.tmpPNGTest, err = fileutil.TempNamedFile("", "", ".png")
	if err != nil {
		t.err = fmt.Errorf("unable to create test PNG: %w", err)
		return
	}

	// We store int of rate*RateFactor so we know we're testing at a set
	// granularity and we have a value that's usable for keying a hash (which
	// floats really aren't)
	var baseRate = int(t.getRate() * RateFactor)

	if t.testRate(baseRate) {
		return
	}

	t.err = fmt.Errorf("could not create a valid JP2")
	return
}

func (t *Transformer) moveTempJP2() {
	// Safety first!
	if t.err != nil {
		return
	}

	t.Logger.Infof("Copying temp JP2 to %s", t.OutputJP2)
	var err = os.Link(t.tmpJP2, t.OutputJP2)
	if err != nil {
		var copyErr = fileutil.CopyVerify(t.tmpJP2, t.OutputJP2)
		if copyErr != nil {
			os.Remove(t.OutputJP2)
			t.err = fmt.Errorf("unable to link or copy JP2: %w / %s", err, copyErr)
			return
		}
	}

	// Make sure the JP2 can be read by non-NCA apps!  The output is very
	// restricted, likely due to temp file security.
	err = os.Chmod(t.OutputJP2, 0644)
	if err != nil {
		os.Remove(t.OutputJP2)
		t.err = fmt.Errorf("unable to set JP2 permissions: %w", err)
	}
}

// testRate is a simple helper to create a JP2 and then try to read it
func (t *Transformer) testRate(rate int) bool {
	// Safety first!
	if t.err != nil {
		return false
	}

	var rateFloat = float64(rate) / RateFactor
	t.makeJP2FromPNG(rateFloat)
	if t.testJP2Decompress() {
		t.Logger.Debugf("Success with rate %g", rateFloat)
		return true
	}
	t.makeJP2FromPNGDashI(rateFloat)
	if t.testJP2Decompress() {
		t.Logger.Debugf("Success with rate %d and -I", rate)
		return true
	}

	t.Logger.Debugf("Failure with rate %d", rate)
	return false
}
