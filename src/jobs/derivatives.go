package jobs

import (
	"config"
	"fileutil"
	"logger"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

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
}

func (md *MakeDerivatives) Process(c *config.Config) bool {
	md.Logger.Debug("Starting make-derivatives job for issue id %d", md.DBIssue.ID)

	// Run our serial operations, failing on the first non-ok response
	var ok = md.RunWhileTrue(
		md.findPDFs,
		md.findTIFFs,
		md.validateSourceFiles,
		md.generateDerivatives,
	)
	if !ok {
		return false
	}

	// The derivatives are generated, so failing to update the workflow doesn't
	// actually mean the operation failed; it just means we have to YELL about
	// the problem
	var err = md.updateIssueWorkflow()
	if err != nil {
		logger.Critical("Unable to update issue (dbid %d) workflow post-derivative-generate: %s", md.DBIssue.ID, err)
	}
	return true
}

// findPDFs builds the list of Alto and JP2 derivative sources
func (md *MakeDerivatives) findPDFs() (ok bool) {
	var pdfs, err = fileutil.FindIf(md.Location, func(i os.FileInfo) bool {
		return pdfFilenameRegex.MatchString(i.Name())
	})

	if err != nil {
		md.Logger.Error("Unable to scan for PDFs: %s", err)
		return false
	}

	if len(pdfs) < 1 {
		md.Logger.Error("No valid PDFs found")
		return false
	}
	md.Logger.Debug("Found PDFs: %#v", pdfs)

	for _, pdf := range pdfs {
		md.AltoDerivativeSources = append(md.AltoDerivativeSources, filepath.Join(md.Location, pdf))
		md.JP2DerivativeSources = append(md.JP2DerivativeSources, filepath.Join(md.Location, pdf))
	}

	return true
}

// findTIFFs looks for any TIFF files in the issue directory.  If a single TIFF
// exists, the JP2 derivative sources list is replaced, as we assume TIFFs to
// always be a superior source format when they're present.
func (md *MakeDerivatives) findTIFFs() (ok bool) {
	var tiffs, err = fileutil.FindIf(md.Location, func(i os.FileInfo) bool {
		return tiffFilenameRegex.MatchString(i.Name())
	})

	if err != nil {
		md.Logger.Error("Unable to scan for TIFFs: %s", err)
		return false
	}

	// Having no TIFF files means it's PDF-only, which is perfectly legitimate
	if len(tiffs) < 1 {
		md.Logger.Debug("No TIFFs present")
		return true
	}
	md.Logger.Debug("Found TIFFs: %#v", tiffs)

	md.JP2DerivativeSources = make([]string, len(tiffs))
	for i, tiff := range tiffs {
		md.JP2DerivativeSources[i] = filepath.Join(md.Location, tiff)
	}

	return true
}

// validateSourceFiles is an attempt to verify sanity again.  Some of these
// checks are redundant, but it's clear that with the complexity of our
// process, more failsafes are better than fewer.
//
//     * There must only be *.pdf or *.tiff files
//     * If there are any *.tiff files, then all *.pdf files must have a matching *.tiff file
func (md *MakeDerivatives) validateSourceFiles() (ok bool) {
	var infos, err = fileutil.ReaddirSorted(md.Location)
	if err != nil {
		md.Logger.Error("Unable to scan all files: %s", err)
		return false
	}

	for _, info := range infos {
		if !tiffFilenameRegex.MatchString(info.Name()) && !pdfFilenameRegex.MatchString(info.Name()) {
			md.Logger.Error("Unexpected file found: %q", info.Name())
			return false
		}
	}

	var alen = len(md.AltoDerivativeSources)
	var jlen = len(md.JP2DerivativeSources)
	if alen != jlen {
		md.Logger.Error("Derivative mismatch: there are %d ALTO sources, but %d JP2 sources", alen, jlen)
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
			md.Logger.Error("Derivative mismatch: ALTO source %q doesn't match JP2 source %q", altoSource, jp2Source)
			return false
		}
	}

	return true
}

func (md *MakeDerivatives) generateDerivatives() (ok bool) {
	// Try to build all derivatives regardless of individual failures
	var derivativeSuccess = true
	for _, file := range md.AltoDerivativeSources {
		if !md.createAltoXML(file) {
			derivativeSuccess = false
		}
	}

	for _, file := range md.JP2DerivativeSources {
		if !md.createJP2(file) {
			derivativeSuccess = false
		}
	}

	// TODO: Consider if we want to keep this long-term.  It's useful for
	// archival purposes since it holds manually-entered metadata, but a database
	// dump may be the proper source.
	if !md.generateMetaJSON() {
		derivativeSuccess = false
	}

	// If a single derivative failed, the operation failed
	return derivativeSuccess
}

func (md *MakeDerivatives) createAltoXML(file string) (ok bool) {
	return false
}

func (md *MakeDerivatives) createJP2(file string) (ok bool) {
	return false
}

func (md *MakeDerivatives) generateMetaJSON() (ok bool) {
	return false
}

func (md *MakeDerivatives) updateIssueWorkflow() error {
	md.DBIssue.HasDerivatives = true
	return md.DBIssue.Save()
}
