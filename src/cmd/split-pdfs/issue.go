package main

import (
	"config"
	"db"
	"fileutil"
	"fmt"
	"io/ioutil"
	"logger"
	"os"
	"path/filepath"
	"regexp"
	"schema"
	"shell"
	"strconv"
)

var splitPageFilenames = regexp.MustCompile(`^seq-(\d+).pdf$`)

// Issue holds a schema issue, db issue, and various bits of specific
// information for splitting a publisher's uploaded issue into PDF/a pages
type Issue struct {
	*schema.Issue
	DBIssue        *db.Issue
	FakeMasterFile string // Where we store the processed, combined PDF
	MasterBackup   string // Where the real master file(s) will eventually live
	TempDir        string // Where we do all page-level processing
	WIPDir         string // Where we copy files after processing
	FinalOutputDir string // Where we move files after the copy was successful
	GhostScript    string // The path to gs for combining the fake master PDF
}

// Dir gives us a single-level directory for the issue in a similar way to the
// schema.Issue.Key() function, but with no path delimiters
func (i *Issue) Dir() string {
	return fmt.Sprintf("%s-%s%02d", i.Title.LCCN, i.DateString(), i.Edition)
}

// ProcessPDFs combines, splits, and then renames files so they're sequential
// in a "best guess" order.  Files are then put into place for manual
// processors to reorder if necessary, remove duped pages, etc.
func (i *Issue) ProcessPDFs(config *config.Config) {
	logger.Debug("Processing issue id %d (%q)", i.DBIssue.ID, i.Key())
	if !i.makeTempFiles() {
		return
	}
	defer i.removeTempFiles()

	i.WIPDir = filepath.Join(config.PDFPageReviewPath, ".wip-"+i.Dir())
	i.FinalOutputDir = filepath.Join(config.PDFPageReviewPath, i.Dir())
	i.MasterBackup = filepath.Join(config.MasterPDFBackupPath, i.Dir())

	if !fileutil.MustNotExist(i.WIPDir) {
		logger.Error("WIP dir %q already exists", i.WIPDir)
		return
	}
	if !fileutil.MustNotExist(i.FinalOutputDir) {
		logger.Error("Final output dir %q already exists", i.FinalOutputDir)
		return
	}
	if !fileutil.MustNotExist(i.MasterBackup) {
		logger.Error("Master backup dir %q already exists", i.MasterBackup)
		return
	}

	i.GhostScript = config.GhostScript
	if i.process() {
		i.DBIssue.Location = i.FinalOutputDir
		i.DBIssue.WorkflowStep = db.WSAwaitingManualProcessing
		var err = i.DBIssue.Save()
		if err != nil {
			logger.Critical("Unable to update workflow metadata after page splitting: %s", err)
		}
	}
}

func (i *Issue) makeTempFiles() (ok bool) {
	var f, err = ioutil.TempFile("", "splitter-master-")
	if err != nil {
		logger.Error("Unable to create temp file for combining PDFs: %s", err)
		return false
	}
	i.FakeMasterFile = f.Name()
	f.Close()

	i.TempDir, err = ioutil.TempDir("", "splitter-pages-")
	if err != nil {
		logger.Error("Unable to create temp dir for issue processing: %s", err)
		return false
	}

	return true
}

func (i *Issue) removeTempFiles() {
	var err = os.Remove(i.FakeMasterFile)
	if err != nil {
		logger.Warn("Unable to remove temp file %q: %s", i.FakeMasterFile, err)
	}
	err = os.RemoveAll(i.TempDir)
	if err != nil {
		logger.Warn("Unable to remove temp dir %q: %s", i.TempDir, err)
	}
}

func (i *Issue) process() (ok bool) {
	if !i.createMasterPDF() {
		return false
	}
	if !i.splitPages() {
		return false
	}
	if !i.fixPageNames() {
		return false
	}
	if !i.convertToPDFA() {
		return false
	}
	if !i.moveToPageReview() {
		return false
	}
	if !i.backupOriginals() {
		return false
	}
	if !i.createMetaJSON() {
		return false
	}

	return true
}

// createMasterPDF combines pages and pre-processes PDFs - ghostscript seems to
// be able to handle some PDFs that crash poppler utils (even as recent as 0.41)
func (i *Issue) createMasterPDF() (ok bool) {
	logger.Debug("Preprocessing with ghostscript")

	var fileinfos, err = fileutil.ReaddirSorted(i.Location)
	if err != nil {
		logger.Error("Unable to list files in %q: %s", i.Location, err)
		return false
	}

	var args = []string{
		"-sDEVICE=pdfwrite", "-dCompatibilityLevel=1.6", "-dPDFSETTINGS=/default",
		"-dNOPAUSE", "-dQUIET", "-dBATCH", "-dDetectDuplicateImages",
		"-dCompressFonts=true", "-r150", "-sOutputFile=" + i.FakeMasterFile,
	}
	for _, fi := range fileinfos {
		args = append(args, filepath.Join(i.Location, fi.Name()))
	}
	return shell.Exec(i.GhostScript, args...)
}

// splitPages ensures we end up with exactly one PDF per page
func (i *Issue) splitPages() (ok bool) {
	logger.Info("Splitting PDF(s)")
	return shell.Exec("pdfseparate", i.FakeMasterFile, filepath.Join(i.TempDir, "seq-%d.pdf"))
}

// fixPageNames converts sequenced PDFs to have 4-digit page numbers
func (i *Issue) fixPageNames() (ok bool) {
	logger.Info("Renaming pages so they're sortable")
	var fileinfos, err = fileutil.ReaddirSorted(i.TempDir)
	if err != nil {
		logger.Error("Unable to read seq-* files for renumbering")
		return false
	}

	for _, fi := range fileinfos {
		var name = fi.Name()
		var fullPath = filepath.Join(i.TempDir, name)
		var matches = splitPageFilenames.FindStringSubmatch(name)
		if len(matches) != 2 || matches[1] == "" {
			logger.Error("File %q doesn't match expected pdf page pattern!", fullPath)
			return false
		}

		var pageNum int
		pageNum, err = strconv.Atoi(matches[1])
		if err != nil {
			logger.Critical("Error parsing pagenum for %q: %s", fullPath, err)
			return false
		}

		var newFullPath = filepath.Join(i.TempDir, fmt.Sprintf("seq-%04d.pdf", pageNum))
		err = os.Rename(fullPath, newFullPath)
		if err != nil {
			logger.Error("Unable to rename %q to %q: %s", fullPath, newFullPath, err)
			return false
		}
	}

	return true
}

// convertToPDFA finds all files in the temp dir and converts them to PDF/a
func (i *Issue) convertToPDFA() (ok bool) {
	logger.Info("Converting pages to PDF/A")
	var fileinfos, err = fileutil.ReaddirSorted(i.TempDir)
	if err != nil {
		logger.Error("Unable to read seq-* files for PDF/a conversion")
		return false
	}

	for _, fi := range fileinfos {
		var fullPath = filepath.Join(i.TempDir, fi.Name())
		logger.Debug("Converting %q to PDF/a", fullPath)
		var dotA = fullPath + ".a"
		var ok = shell.Exec(i.GhostScript, "-dPDFA=2", "-dBATCH", "-dNOPAUSE",
			"-sProcessColorModel=DeviceCMYK", "-sDEVICE=pdfwrite",
			"-sPDFACompatibilityPolicy=1", "-sOutputFile="+dotA, fullPath)
		if !ok {
			return false
		}

		err = os.Rename(fullPath+".a", fullPath)
		if err != nil {
			logger.Error("Unable to rename PDF/a file %q to %q: %s", dotA, fullPath, err)
			return false
		}
	}

	return true
}

// moveToPageReview copies tmpdir to the WIPDir, then moves it to the final
// location once the copy succeeded so we can avoid broken dir moves
func (i *Issue) moveToPageReview() (ok bool) {
	var err = fileutil.CopyDirectory(i.TempDir, i.WIPDir)
	if err != nil {
		logger.Error("Unable to move temporary directory %q to %q", i.TempDir, i.WIPDir)
		return false
	}
	err = os.Rename(i.WIPDir, i.FinalOutputDir)
	if err != nil {
		logger.Error("Unable to rename WIP directory %q to %q", i.WIPDir, i.FinalOutputDir)
		return false
	}

	return true
}

// backupOriginals stores the original uploads in the master backup location.
// If this fails, we have a problem, because the pages were already split and
// moved.  All we can do is log critical errors.
func (i *Issue) backupOriginals() (ok bool) {
	var masterParent = filepath.Dir(i.MasterBackup)
	var err = os.MkdirAll(masterParent, 0700)
	if err != nil {
		logger.Critical("Unable to create master backup parent %q: %s", masterParent, err)
		return false
	}

	err = fileutil.CopyDirectory(i.Location, i.MasterBackup)
	if err != nil {
		logger.Critical("Unable to copy master file(s) from %q to %q: %s", i.Location, i.MasterBackup, err)
		return false
	}

	err = os.RemoveAll(i.Location)
	if err != nil {
		logger.Critical("Unable to remove original files after making master backup: %s", err)
		return false
	}

	return true
}

// createMetaJSON builds and writes out a basic metadata file for legacy
// processors to use
func (i *Issue) createMetaJSON() (ok bool) {
	logger.Warn("Not implemented!")
	return false
}
