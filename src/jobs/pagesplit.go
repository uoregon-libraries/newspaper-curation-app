package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/shell"
)

var splitPageFilenames = regexp.MustCompile(`^seq-(\d+).pdf$`)

// PageSplit is an IssueJob with job-specific information and logic for
// splitting a publisher's uploaded issue into PDF/a pages
type PageSplit struct {
	*IssueJob
	CombinedFile string // Where we store the processed, combined PDF
	TempDir      string // Where we do all page-level processing
	OutputDir    string // Where we copy files after processing
	GhostScript  string // The path to gs for combining the PDF
	PDFSeparate  string // The path to the `pdfseparate` binary for page-splitting
	MinPages     int    // Number of pages below which we refuse to process
}

// Process combines, splits, and then renames files so they're sequential in a
// "best guess" order.  Files are then put into place for manual processors to
// reorder if necessary, remove duped pages, etc.
func (ps *PageSplit) Process(conf *config.Config) ProcessResponse {
	ps.Logger.Debugf("Processing issue id %d (%q)", ps.DBIssue.ID, ps.Issue.Key())
	if !ps.makeTempFiles() {
		return PRFailure
	}
	defer ps.removeTempFiles()

	ps.OutputDir = ps.db.Args[JobArgLocation]
	if !fileutil.MustNotExist(ps.OutputDir) {
		ps.Logger.Errorf("Output dir %q already exists", ps.OutputDir)
		return PRFatal
	}

	ps.GhostScript = conf.GhostScript
	ps.PDFSeparate = conf.PDFSeparate
	ps.MinPages = conf.MinimumIssuePages
	if ps.process() {
		return PRSuccess
	}
	return PRFailure
}

func (ps *PageSplit) makeTempFiles() (ok bool) {
	var err error
	ps.CombinedFile, err = fileutil.TempNamedFile("", "splitter-combined-", ".pdf")
	if err != nil {
		ps.Logger.Errorf("Unable to create temp file for combining PDFs: %s", err)
		return false
	}

	ps.TempDir, err = os.MkdirTemp("", "splitter-pages-")
	if err != nil {
		ps.Logger.Errorf("Unable to create temp dir for issue processing: %s", err)
		return false
	}

	return true
}

func (ps *PageSplit) removeTempFiles() {
	var err = os.Remove(ps.CombinedFile)
	if err != nil {
		ps.Logger.Warnf("Unable to remove temp file %q: %s", ps.CombinedFile, err)
	}
	err = os.RemoveAll(ps.TempDir)
	if err != nil {
		ps.Logger.Warnf("Unable to remove temp dir %q: %s", ps.TempDir, err)
	}
}

func (ps *PageSplit) process() (ok bool) {
	return RunWhileTrue(
		ps.combinePDF,
		ps.splitPages,
		ps.fixPageNames,
		ps.convertToPDFA,
		ps.moveIssue,
	)
}

// combinePDF combines pages and pre-processes PDFs - ghostscript seems to
// be able to handle some PDFs that crash poppler utils (even as recent as 0.41)
func (ps *PageSplit) combinePDF() (ok bool) {
	ps.Logger.Debugf("Preprocessing with ghostscript")

	// Using our custom numeric sort gives us a tiny chance that publishers who
	// upload weirdly-numbered pages won't get put out of order.  In most cases
	// it probably won't help, because their numbers are in the middle of the
	// filename in hard-to-predict ways, but it's worth a try.
	var fileinfos, err = fileutil.ReaddirSortedNumeric(ps.DBIssue.Location)
	if err != nil {
		ps.Logger.Errorf("Unable to list files in %q: %s", ps.DBIssue.Location, err)
		return false
	}

	var args = []string{
		"-sDEVICE=pdfwrite", "-dCompatibilityLevel=1.6", "-dPDFSETTINGS=/default",
		"-dNOPAUSE", "-dQUIET", "-dBATCH", "-dDetectDuplicateImages",
		"-dCompressFonts=true", "-r150", "-sOutputFile=" + ps.CombinedFile,
	}
	for _, fi := range fileinfos {
		args = append(args, filepath.Join(ps.DBIssue.Location, fi.Name()))
	}
	return shell.ExecSubgroup(ps.GhostScript, ps.Logger, args...)
}

// splitPages ensures we end up with exactly one PDF per page
func (ps *PageSplit) splitPages() (ok bool) {
	ps.Logger.Infof("Splitting PDF(s)")
	return shell.ExecSubgroup(ps.PDFSeparate, ps.Logger, ps.CombinedFile, filepath.Join(ps.TempDir, "seq-%d.pdf"))
}

// fixPageNames converts sequenced PDFs to have 4-digit page numbers
func (ps *PageSplit) fixPageNames() (ok bool) {
	ps.Logger.Infof("Renaming pages so they're sortable")
	var fileinfos, err = fileutil.ReaddirSortedNumeric(ps.TempDir)
	if err != nil {
		ps.Logger.Errorf("Unable to read seq-* files for renumbering")
		return false
	}

	if len(fileinfos) < ps.MinPages {
		ps.Logger.Errorf("Too few pages to continue processing (found %d, need %d or more)", len(fileinfos), ps.MinPages)
		return false
	}

	for _, fi := range fileinfos {
		var name = fi.Name()
		var fullPath = filepath.Join(ps.TempDir, name)
		var matches = splitPageFilenames.FindStringSubmatch(name)
		if len(matches) != 2 || matches[1] == "" {
			ps.Logger.Errorf("File %q doesn't match expected pdf page pattern!", fullPath)
			return false
		}

		var pageNum int
		pageNum, err = strconv.Atoi(matches[1])
		if err != nil {
			ps.Logger.Criticalf("Error parsing pagenum for %q: %s", fullPath, err)
			return false
		}

		var newFullPath = filepath.Join(ps.TempDir, fmt.Sprintf("seq-%04d.pdf", pageNum))
		err = os.Rename(fullPath, newFullPath)
		if err != nil {
			ps.Logger.Errorf("Unable to rename %q to %q: %s", fullPath, newFullPath, err)
			return false
		}
	}

	return true
}

// convertToPDFA finds all files in the temp dir and converts them to PDF/a
func (ps *PageSplit) convertToPDFA() (ok bool) {
	ps.Logger.Infof("Converting pages to PDF/A")
	var fileinfos, err = fileutil.ReaddirSortedNumeric(ps.TempDir)
	if err != nil {
		ps.Logger.Errorf("Unable to read seq-* files for PDF/a conversion")
		return false
	}

	for _, fi := range fileinfos {
		var fullPath = filepath.Join(ps.TempDir, fi.Name())
		ps.Logger.Debugf("Converting %q to PDF/a", fullPath)
		var dotA = fullPath + ".a"
		var ok = shell.ExecSubgroup(ps.GhostScript, ps.Logger, "-dPDFA=2", "-dBATCH", "-dNOPAUSE",
			"-sProcessColorModel=DeviceCMYK", "-sDEVICE=pdfwrite",
			"-sPDFACompatibilityPolicy=1", "-sOutputFile="+dotA, fullPath)
		if !ok {
			return false
		}

		err = os.Rename(fullPath+".a", fullPath)
		if err != nil {
			ps.Logger.Errorf("Unable to rename PDF/a file %q to %q: %s", dotA, fullPath, err)
			return false
		}
	}

	return true
}

// moveIssue moves the processed files into the "final" output directory
func (ps *PageSplit) moveIssue() (ok bool) {
	var err = fileutil.CopyDirectory(ps.TempDir, ps.OutputDir)
	if err != nil {
		ps.Logger.Errorf("Unable to move temporary directory %q to %q: %s", ps.TempDir, ps.OutputDir, err)
		return false
	}

	// Make sure RAIS, Apache, etc. can read the copied dir
	err = os.Chmod(ps.OutputDir, 0755)
	if err != nil {
		ps.Logger.Errorf("Unable to change permissions on %q: %s", ps.OutputDir, err)
		return false
	}

	return true
}
