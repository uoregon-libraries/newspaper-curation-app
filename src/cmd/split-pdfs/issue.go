package main

import (
	"config"
	"db"
	"fileutil"
	"io/ioutil"
	"logger"
	"os"
	"path/filepath"
	"regexp"
	"schema"
	"shell"
)

var splitPageFilenames = regexp.MustCompile(`^.*/seq-(\d+).pdf$`)

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
	var f, err = ioutil.TempFile("", "")
	if err != nil {
		logger.Error("Unable to create temp file for combining PDFs: %s", err)
		return false
	}
	i.FakeMasterFile = f.Name()
	f.Close()

	i.TempDir, err = ioutil.TempDir("", "")
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
		return false
	}

	var args = []string{
		"-sDEVICE=pdfwrite", "-dCompatibilityLevel=1.6", "-dPDFSETTINGS=/default",
		"-dNOPAUSE", "-dQUIET", "-dBATCH", "-dDetectDuplicateImages",
		"-dCompressFonts=true", "-r150", "-sOutputFile=" + i.FakeMasterFile,
	}
	for _, fi := range fileinfos {
		args = append(args, fi.Name())
	}
	return shell.Exec(i.GhostScript, args...)
}

// splitPages ensures we end up with exactly one PDF per page
func (i *Issue) splitPages() (ok bool) {
	logger.Info("Splitting PDF(s)")
	return shell.Exec("pdfseparate", i.FakeMasterFile, filepath.Join(i.TempDir, "seq-%d.pdf"))
}

/*
  def fix_page_names(self, directory):
    """Convert sequenced PDFs to have 4-digit page numbers"""
    self.log.info("Renaming pages so they're sortable")
    p = re.compile(splitPageFilenames)
    for pdfpage in utils.find(directory, "seq-*.pdf"):
      self.log.debug("Renaming %s to be sortable" % pdfpage)
      m = p.match(pdfpage)
      if m:
        shutil.move(pdfpage, os.path.join(directory, "seq-%04d.pdf" % int(m.group(1))))
      else:
        self.log.error("File '%s' didn't match expected pdf page pattern!" % pdfpage)

  def convert_to_pdfa(self, tempdir):
    self.log.info("Converting pages to PDF/A")
    for pdfpage in utils.find(tempdir, "seq-*.pdf"):
      self.log.debug("PDF-A for %s" % pdfpage)
      exit_status = utils.shell([settings.GHOSTSCRIPT, "-dPDFA=2", "-dBATCH", "-dNOPAUSE", "-sProcessColorModel=DeviceCMYK",
          "-sDEVICE=pdfwrite", "-sPDFACompatibilityPolicy=1",
          "-sOutputFile=%s.a" % pdfpage, pdfpage])
      if exit_status != 0:
        return False

      shutil.move("%s.a" % pdfpage, pdfpage)

    return True

  def process_issue(self, pdf_dir, tempdir):
    self.fix_page_names(tempdir)

    if self.convert_to_pdfa(tempdir) != True:
      self.log.error("Unable to convert pages to PDF/A")
      return

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
