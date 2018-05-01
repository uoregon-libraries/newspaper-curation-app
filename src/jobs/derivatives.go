package jobs

import (
	"config"
	"derivatives/alto"
	"derivatives/jp2"

	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

var allowedFilesRegex = regexp.MustCompile(`(?i:^([0-9]{4}.(pdf|jp2|xml|tiff?))|[0-9]{10}.xml|master.tar)`)
var pdfFilenameRegex = regexp.MustCompile(`(?i:^[0-9]{4}.pdf)`)
var tiffFilenameRegex = regexp.MustCompile(`(?i:^[0-9]{4}.tiff?)`)

// MakeDerivatives is a job which creates all necessary derivatives for a given
// issue, detecting whether PDFs are needed and whether JP2s should be build
// from PDF or TIFF sources.  Derivatives are built independently, and get
// placed directly into the issue's existing path, so this job is very
// requeue-friendly if just a few files are broken / missing.
type MakeDerivatives struct {
	*IssueJob
	AltoDerivativeSources []string
	JP2DerivativeSources  []string
	findTIFFs             func() bool
	AltoDPI               int
	JP2DPI                int
	JP2Quality            float64
	OPJCompress           string
	OPJDecompress         string
	GhostScript           string
}

// Process generates the derivatives for the job's issue
func (md *MakeDerivatives) Process(c *config.Config) bool {
	md.Logger.Debugf("Starting make-derivatives job for issue id %d", md.DBIssue.ID)

	md.updateWorkflowCB = md.updateIssueWorkflow
	md.OPJCompress = c.OPJCompress
	md.OPJDecompress = c.OPJDecompress
	md.GhostScript = c.GhostScript
	md.JP2DPI = c.DPI
	md.JP2Quality = c.Quality

	if md.DBIssue.IsFromScanner {
		// For scanned issues, we have to verify TIFFs and use the scan DPI for
		// generating ALTO XML
		md.findTIFFs = md._findTIFFs
		md.AltoDPI = c.ScannedPDFDPI
	} else {
		// Born-digital issues don't check TIFFs and use the JP2 DPI for ALTO
		md.findTIFFs = func() bool { return true }
		md.AltoDPI = c.DPI
	}

	// Run our serial operations, failing on the first non-ok response
	return RunWhileTrue(
		md.findPDFs,
		md.findTIFFs,
		md.validateSourceFiles,
		md.generateDerivatives,
	)
}

// findPDFs builds the list of Alto and JP2 derivative sources
func (md *MakeDerivatives) findPDFs() (ok bool) {
	var pdfs, err = fileutil.FindIf(md.Location, func(i os.FileInfo) bool {
		return pdfFilenameRegex.MatchString(i.Name())
	})

	if err != nil {
		md.Logger.Errorf("Unable to scan for PDFs: %s", err)
		return false
	}

	if len(pdfs) < 1 {
		md.Logger.Errorf("No valid PDFs found")
		return false
	}
	md.Logger.Debugf("Found %d PDFs", len(pdfs))

	for _, pdf := range pdfs {
		md.AltoDerivativeSources = append(md.AltoDerivativeSources, pdf)
		md.JP2DerivativeSources = append(md.JP2DerivativeSources, pdf)
	}

	return true
}

// _findTIFFs looks for any TIFF files in the issue directory.  This is only
// called for scanned issues, so there *must* be TIFFs or this is a failure.
func (md *MakeDerivatives) _findTIFFs() (ok bool) {
	var tiffs, err = fileutil.FindIf(md.Location, func(i os.FileInfo) bool {
		return tiffFilenameRegex.MatchString(i.Name())
	})

	if err != nil {
		md.Logger.Errorf("Unable to scan for TIFFs: %s", err)
		return false
	}

	if len(tiffs) < 1 {
		md.Logger.Errorf("No TIFFs present")
		return false
	}
	md.Logger.Debugf("Found %d TIFFs", len(tiffs))

	md.JP2DerivativeSources = make([]string, len(tiffs))
	for i, tiff := range tiffs {
		md.JP2DerivativeSources[i] = tiff
	}

	return true
}

// validateSourceFiles is an attempt to verify sanity again.  Some of these
// checks are redundant, but it's clear that with the complexity of our
// process, more failsafes are better than fewer.
func (md *MakeDerivatives) validateSourceFiles() (ok bool) {
	var infos, err = fileutil.ReaddirSorted(md.Location)
	if err != nil {
		md.Logger.Errorf("Unable to scan all files: %s", err)
		return false
	}

	for _, info := range infos {
		var n = info.Name()
		if !allowedFilesRegex.MatchString(n) {
			md.Logger.Errorf("Unexpected file found: %q", n)
			return false
		}
	}

	var alen = len(md.AltoDerivativeSources)
	var jlen = len(md.JP2DerivativeSources)
	if alen != jlen {
		md.Logger.Errorf("Derivative mismatch: there are %d ALTO sources, but %d JP2 sources", alen, jlen)
		return false
	}

	for i, altoSource := range md.AltoDerivativeSources {
		var jp2Source = md.JP2DerivativeSources[i]
		var altoBase = filepath.Base(altoSource)
		var jp2Base = filepath.Base(jp2Source)
		var altoParts = strings.Split(altoBase, ".")
		var altoNoExt = altoParts[0]
		var jp2Parts = strings.Split(jp2Base, ".")
		var jp2NoExt = jp2Parts[0]
		if altoNoExt != jp2NoExt {
			md.Logger.Errorf("Derivative mismatch: At index %d, ALTO source (%q) doesn't match JP2 source (%q)",
				i, altoSource, jp2Source)
			return false
		}
	}

	return true
}

func (md *MakeDerivatives) generateDerivatives() (ok bool) {
	// Try to build all derivatives regardless of individual failures
	ok = true
	for i, file := range md.AltoDerivativeSources {
		ok = ok && md.createAltoXML(file, i+1)
	}

	for _, file := range md.JP2DerivativeSources {
		ok = ok && md.createJP2(file)
	}

	// If a single derivative failed, the operation failed
	return ok
}

// createAltoXML produces ALTO XML from the given PDF file
func (md *MakeDerivatives) createAltoXML(file string, pageno int) (ok bool) {
	var outputFile = strings.Replace(file, filepath.Ext(file), ".xml", 1)
	var transformer = alto.New(file, outputFile, md.AltoDPI, pageno)
	transformer.Logger = md.Logger
	var err = transformer.Transform()

	if err != nil {
		md.Logger.Errorf("Couldn't convert %q to ALTO: %s", file, err)
		return false
	}

	return true
}

func (md *MakeDerivatives) createJP2(file string) (ok bool) {
	var outputJP2 = strings.Replace(file, filepath.Ext(file), ".jp2", 1)
	var transformer = jp2.New(file, outputJP2, md.JP2Quality, md.JP2DPI)
	transformer.Logger = md.Logger
	transformer.OPJCompress = md.OPJCompress
	transformer.OPJDecompress = md.OPJDecompress
	transformer.GhostScript = md.GhostScript

	var err = transformer.Transform()
	if err != nil {
		md.Logger.Errorf("Couldn't convert %q to JP2: %s", file, err)
		return false
	}

	return true
}

func (md *MakeDerivatives) updateIssueWorkflow() {
	md.DBIssue.HasDerivatives = true
}
