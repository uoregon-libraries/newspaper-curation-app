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
	TempDir        string // Where we do all page-level processing
	OutputDir      string // Where we put files when they're considered complete
	GhostScript    string // The path to gs for combining the fake master PDF
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

	i.OutputDir = config.PDFPageReviewPath
	i.GhostScript = config.GhostScript
	i.process()
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

func (i *Issue) process() {
	if !i.createMasterPDF() {
		return
	}
	if !i.splitPages() {
		return
	}
	if !i.fixPageNames() {
		return
	}
	if !i.convertToPDFA() {
		return
	}

	// Copy tmpdir to "<page review>/.wip/<issue dir>", then move it once the
	// copy succeeded so we can avoid broken dir moves
	// TODO: Copy tmpdir -> config.PDFPageReviewPath/.wip/issuekey

	// Copy the original file(s) into a "-wip" folder, remove the original, and
	// then rename the "-wip" folder
	// TODO: copy to config.MasterPDFBackupPath
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

/*
  def process_issue(self, pdf_dir, tempdir):
    self.log.info("Moving split pages to '%s'" % pdf_dir.pdf_split_dir)
    os.makedirs(pdf_dir.pdf_split_dir)
    for pdfpage in utils.find(tempdir, "seq-*.pdf"):
      shutil.move(pdfpage, pdf_dir.pdf_split_dir)

    self.log.info("Tagging file hashes")
    utils.tag_pdf_hashes(pdf_dir.pdf_split_dir)

    self.log.info("Backing up to '%s' and cleaning up" % pdf_dir.master_backup)
    d, f = os.path.split(pdf_dir.master_backup)
    if not os.path.exists(d):
      os.makedirs(d)
    shutil.move(pdf_dir.full_path, pdf_dir.master_backup)

    self.log.info("Storing generated path to issue for linking backup")
    metafile = "%s/.meta.json" % pdf_dir.pdf_split_dir
    utils.buildmeta(metafile, pdf_dir.structured_subpath, settings.PDF_BATCH_MARC_ORG_CODE)
*/
