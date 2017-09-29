// Package jp2 converts a PDF or TIFF into a JP2.  The resulting JP2 is then
// verified as being readable and re-encoded at various rate values if not.
// This is a huge hack to try and centralize the combination of PDF- and
// TIFF-to-JP2 conversion, testing, and fixing.
package jp2

import (
	"fileutil"
	"fmt"
	"logger"
	"os"
	"path/filepath"
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

	err         error
	testedRates map[int]bool
	Logger      logger.Logger
}

// New creates a new PDF/TIFF-to-JP2 transformer with default values for the
// various binaries and use of the default logger
func New(source, output string, quality float64, resolution int) *Transformer {
	return &Transformer{
		SourceFile:     source,
		OutputJP2:      output,
		OPJCompress:    "opj_compress",
		OPJDecompress:  "opj_decompress",
		GhostScript:    "gs",
		GraphicsMagick: "gm",
		Quality:        quality,
		PDFResolution:  resolution,
		Logger:         logger.DefaultLogger,
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
		t.Logger.Info("Not generating JP2 file %q; file already exists", t.OutputJP2)
		return nil
	}

	t.makePNG()
	t.makeJP2()
	t.moveTempJP2()

	t.Logger.Debug("Removing tmpPNG %q", t.tmpPNG)
	os.Remove(t.tmpPNG)
	t.Logger.Debug("Removing tmpJP2 %q", t.tmpJP2)
	os.Remove(t.tmpJP2)
	t.Logger.Debug("Removing tmpPNGTest %q", t.tmpPNGTest)
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

	t.Logger.Info("Creating PNG from %q", t.SourceFile)

	t.tmpPNG, err = fileutil.TempNamedFile("", "", ".png")
	if err != nil {
		t.err = fmt.Errorf("unable to create temporary PNG: %s", err)
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

	t.Logger.Info("Creating JP2 from PNG")

	// Create a temp file for holding our JP2
	t.tmpJP2, err = fileutil.TempNamedFile("", "", ".jp2")
	if err != nil {
		t.err = fmt.Errorf("unable to create test JP2: %s", err)
		return
	}

	// Create a stable test PNG for use in the JP2 decode verification(s)
	t.tmpPNGTest, err = fileutil.TempNamedFile("", "", ".png")
	if err != nil {
		t.err = fmt.Errorf("unable to create test PNG: %s", err)
		return
	}

	// We store int of rate*RateFactor so we know we're testing at a set
	// granularity and we have a value that's usable for keying a hash (which
	// floats really aren't)
	var baseRate = int(t.getRate() * RateFactor)

	var rangeQueue = &RangeQueue{}
	t.testedRates = make(map[int]bool)

	// First "range" is just the base rate since that's the ideal value
	rangeQueue.Append(baseRate, baseRate)

	// 2/3 * x to try something semi-distant but not super expensive in terms of storage
	rangeQueue.Append(baseRate*2/3, baseRate)

	for i := 0; len(rangeQueue.queue) > 0; i++ {
		var r = rangeQueue.Shift()
		if r == EmptyRange {
			continue
		}

		if t.testRate(r.start) {
			return
		}
		if t.testRate(r.end) {
			return
		}

		var midPoint = (r.start + r.end) / 2
		rangeQueue.Append(r.start, midPoint)
		rangeQueue.Append(midPoint, r.end)

		// After a while we are willing to lose a little quality
		if i == 5 {
			rangeQueue.Append(baseRate, baseRate*5/4)
		}

		// Later on, we expand the search further.  This can cost a good deal of
		// space, but we're getting desperate.
		if i == 50 {
			rangeQueue.Append(baseRate/3, baseRate*2/3)
		}

		// If we had no successes after exhausting all those options above, we're
		// willing to sacrifice a lot more quality or space
		if i > 50 && len(rangeQueue.queue) == 0 {
			rangeQueue.Append(baseRate/6, baseRate/3)
			rangeQueue.Append(baseRate*5/4, baseRate*3/2)
		}
	}

	t.err = fmt.Errorf("no rate found for creating a valid JP2")
	return
}

func (t *Transformer) moveTempJP2() {
	// Safety first!
	if t.err != nil {
		return
	}

	t.Logger.Info("Copying temp JP2 to %s", t.OutputJP2)
	var err = os.Link(t.tmpJP2, t.OutputJP2)
	if err != nil {
		var copyErr = fileutil.CopyFile(t.tmpJP2, t.OutputJP2)
		if copyErr != nil {
			t.err = fmt.Errorf("unable to link or copy JP2: %s / %s", err, copyErr)
			return
		}
	}
}

// testRate is a simple helper to create a JP2 and then try to read it
func (t *Transformer) testRate(rate int) bool {
	// Safety first!
	if t.err != nil {
		return false
	}

	var rateFloat = float64(rate) / RateFactor
	if t.testedRates[rate] {
		t.Logger.Debug("Skipping already-tested rate %g", rate)
		return false
	}
	t.testedRates[rate] = true

	t.makeJP2FromPNG(rateFloat)
	if t.testJP2Decompress() {
		t.Logger.Debug("Success with rate %g", rateFloat)
		return true
	}
	t.makeJP2FromPNGDashI(rateFloat)
	if t.testJP2Decompress() {
		t.Logger.Debug("Success with rate %g and -I", rate)
		return true
	}

	t.Logger.Debug("Failure with rate %g", rate)
	return false
}
