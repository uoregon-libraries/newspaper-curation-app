package jobs

import (
	"config"
	"fileutil"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"shell"
	"strconv"
)

var splitPageFilenames = regexp.MustCompile(`^seq-(\d+).pdf$`)

// PageSplit is an IssueJob with job-specific information and logic for
// splitting a publisher's uploaded issue into PDF/a pages
type PageSplit struct {
	*IssueJob
	FakeMasterFile string // Where we store the processed, combined PDF
	MasterBackup   string // Where the real master file(s) will eventually live
	TempDir        string // Where we do all page-level processing
	WIPDir         string // Where we copy files after processing
	FinalOutputDir string // Where we move files after the copy was successful
	GhostScript    string // The path to gs for combining the fake master PDF
}

// Dir gives us a single-level directory for the issue in a similar way to the
// schema.Issue.Key() function, but with no path delimiters
func (ps *PageSplit) Dir() string {
	return fmt.Sprintf("%s-%s%02d", ps.Issue.Title.LCCN, ps.Issue.DateString(), ps.Issue.Edition)
}

// Process combines, splits, and then renames files so they're sequential in a
// "best guess" order.  Files are then put into place for manual processors to
// reorder if necessary, remove duped pages, etc.
func (ps *PageSplit) Process(config *config.Config) bool {
	ps.Logger.Debug("Processing issue id %d (%q)", ps.DBIssue.ID, ps.Issue.Key())
	if !ps.makeTempFiles() {
		return false
	}
	defer ps.removeTempFiles()

	ps.WIPDir = filepath.Join(config.PDFPageReviewPath, ".wip-"+ps.Dir())
	ps.FinalOutputDir = filepath.Join(config.PDFPageReviewPath, ps.Dir())
	ps.MasterBackup = filepath.Join(config.MasterPDFBackupPath, ps.Dir())

	if !fileutil.MustNotExist(ps.WIPDir) {
		ps.Logger.Error("WIP dir %q already exists", ps.WIPDir)
		return false
	}
	if !fileutil.MustNotExist(ps.FinalOutputDir) {
		ps.Logger.Error("Final output dir %q already exists", ps.FinalOutputDir)
		return false
	}
	if !fileutil.MustNotExist(ps.MasterBackup) {
		ps.Logger.Error("Master backup dir %q already exists", ps.MasterBackup)
		return false
	}

	ps.GhostScript = config.GhostScript
	return ps.process()
}

func (ps *PageSplit) makeTempFiles() (ok bool) {
	var f, err = ioutil.TempFile("", "splitter-master-")
	if err != nil {
		ps.Logger.Error("Unable to create temp file for combining PDFs: %s", err)
		return false
	}
	ps.FakeMasterFile = f.Name()
	f.Close()

	ps.TempDir, err = ioutil.TempDir("", "splitter-pages-")
	if err != nil {
		ps.Logger.Error("Unable to create temp dir for issue processing: %s", err)
		return false
	}

	return true
}

func (ps *PageSplit) removeTempFiles() {
	var err = os.Remove(ps.FakeMasterFile)
	if err != nil {
		ps.Logger.Warn("Unable to remove temp file %q: %s", ps.FakeMasterFile, err)
	}
	err = os.RemoveAll(ps.TempDir)
	if err != nil {
		ps.Logger.Warn("Unable to remove temp dir %q: %s", ps.TempDir, err)
	}
}

func (ps *PageSplit) process() (ok bool) {
	if !ps.createMasterPDF() {
		return false
	}
	if !ps.splitPages() {
		return false
	}
	if !ps.fixPageNames() {
		return false
	}
	if !ps.convertToPDFA() {
		return false
	}
	if !ps.backupOriginals() {
		return false
	}
	return ps.moveToPageReview()
}

// createMasterPDF combines pages and pre-processes PDFs - ghostscript seems to
// be able to handle some PDFs that crash poppler utils (even as recent as 0.41)
func (ps *PageSplit) createMasterPDF() (ok bool) {
	ps.Logger.Debug("Preprocessing with ghostscript")

	var fileinfos, err = fileutil.ReaddirSorted(ps.Location)
	if err != nil {
		ps.Logger.Error("Unable to list files in %q: %s", ps.Location, err)
		return false
	}

	var args = []string{
		"-sDEVICE=pdfwrite", "-dCompatibilityLevel=1.6", "-dPDFSETTINGS=/default",
		"-dNOPAUSE", "-dQUIET", "-dBATCH", "-dDetectDuplicateImages",
		"-dCompressFonts=true", "-r150", "-sOutputFile=" + ps.FakeMasterFile,
	}
	for _, fi := range fileinfos {
		args = append(args, filepath.Join(ps.Location, fi.Name()))
	}
	return shell.Exec(ps.GhostScript, args...)
}

// splitPages ensures we end up with exactly one PDF per page
func (ps *PageSplit) splitPages() (ok bool) {
	ps.Logger.Info("Splitting PDF(s)")
	return shell.Exec("pdfseparate", ps.FakeMasterFile, filepath.Join(ps.TempDir, "seq-%d.pdf"))
}

// fixPageNames converts sequenced PDFs to have 4-digit page numbers
func (ps *PageSplit) fixPageNames() (ok bool) {
	ps.Logger.Info("Renaming pages so they're sortable")
	var fileinfos, err = fileutil.ReaddirSorted(ps.TempDir)
	if err != nil {
		ps.Logger.Error("Unable to read seq-* files for renumbering")
		return false
	}

	for _, fi := range fileinfos {
		var name = fi.Name()
		var fullPath = filepath.Join(ps.TempDir, name)
		var matches = splitPageFilenames.FindStringSubmatch(name)
		if len(matches) != 2 || matches[1] == "" {
			ps.Logger.Error("File %q doesn't match expected pdf page pattern!", fullPath)
			return false
		}

		var pageNum int
		pageNum, err = strconv.Atoi(matches[1])
		if err != nil {
			ps.Logger.Critical("Error parsing pagenum for %q: %s", fullPath, err)
			return false
		}

		var newFullPath = filepath.Join(ps.TempDir, fmt.Sprintf("seq-%04d.pdf", pageNum))
		err = os.Rename(fullPath, newFullPath)
		if err != nil {
			ps.Logger.Error("Unable to rename %q to %q: %s", fullPath, newFullPath, err)
			return false
		}
	}

	return true
}

// convertToPDFA finds all files in the temp dir and converts them to PDF/a
func (ps *PageSplit) convertToPDFA() (ok bool) {
	ps.Logger.Info("Converting pages to PDF/A")
	var fileinfos, err = fileutil.ReaddirSorted(ps.TempDir)
	if err != nil {
		ps.Logger.Error("Unable to read seq-* files for PDF/a conversion")
		return false
	}

	for _, fi := range fileinfos {
		var fullPath = filepath.Join(ps.TempDir, fi.Name())
		ps.Logger.Debug("Converting %q to PDF/a", fullPath)
		var dotA = fullPath + ".a"
		var ok = shell.Exec(ps.GhostScript, "-dPDFA=2", "-dBATCH", "-dNOPAUSE",
			"-sProcessColorModel=DeviceCMYK", "-sDEVICE=pdfwrite",
			"-sPDFACompatibilityPolicy=1", "-sOutputFile="+dotA, fullPath)
		if !ok {
			return false
		}

		err = os.Rename(fullPath+".a", fullPath)
		if err != nil {
			ps.Logger.Error("Unable to rename PDF/a file %q to %q: %s", dotA, fullPath, err)
			return false
		}
	}

	return true
}

// moveToPageReview copies tmpdir to the WIPDir, then moves it to the final
// location once the copy succeeded so we can avoid broken dir moves
func (ps *PageSplit) moveToPageReview() (ok bool) {
	var err = fileutil.CopyDirectory(ps.TempDir, ps.WIPDir)
	if err != nil {
		ps.Logger.Error("Unable to move temporary directory %q to %q", ps.TempDir, ps.WIPDir)
		return false
	}
	err = os.Rename(ps.WIPDir, ps.FinalOutputDir)
	if err != nil {
		ps.Logger.Error("Unable to rename WIP directory %q to %q", ps.WIPDir, ps.FinalOutputDir)
		return false
	}

	return true
}

// backupOriginals stores the original uploads in the master backup location.
// If this fails, we have a problem, because the pages were already split and
// moved.  All we can do is log critical errors.
func (ps *PageSplit) backupOriginals() (ok bool) {
	var masterParent = filepath.Dir(ps.MasterBackup)
	var err = os.MkdirAll(masterParent, 0700)
	if err != nil {
		ps.Logger.Critical("Unable to create master backup parent %q: %s", masterParent, err)
		return false
	}

	err = fileutil.CopyDirectory(ps.Location, ps.MasterBackup)
	if err != nil {
		ps.Logger.Critical("Unable to copy master file(s) from %q to %q: %s", ps.Location, ps.MasterBackup, err)
		return false
	}

	err = os.RemoveAll(ps.Location)
	if err != nil {
		ps.Logger.Critical("Unable to remove original files after making master backup: %s", err)
		return false
	}

	return true
}