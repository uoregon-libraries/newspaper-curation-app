package main

import (
	"config"
	"db"
	"io/ioutil"
	"logger"
	"os"
	"regexp"
	"schema"
)

var splitPageFilenames = regexp.MustCompile(`^.*/seq-(\d+).pdf$`)

// Issue wraps a schema issue and hold the DB issue to allow file processing
// after a DB lookup
type Issue struct {
	*schema.Issue
	DBIssue *db.Issue
}

// ProcessPDFs combines, splits, and then renames files so they're sequential
// in a "best guess" order.  Files are then put into place for manual
// processors to reorder if necessary, remove duped pages, etc.
func (i *Issue) ProcessPDFs(config *config.Config) {
	var tmpdir, err = ioutil.TempDir("", "")
	if err != nil {
		logger.Error("Unable to create temp dir for issue processing: %s", err)
		return
	}
	logger.Debug("Processing issue id %d (%q)", i.DBIssue.ID, i.Key())
	i.process(config, tmpdir)
	err = os.RemoveAll(tmpdir)
	if err != nil {
		logger.Warn("Unable to remove temp dir %q: %s", tmpdir, err)
	}
}

func (i *Issue) process(config *config.Config, tmpdir string) {
	// TODO: Combine issue PDFs with output going to tmpdir
	// TODO: Split the fake-master PDF in tmpdir

	// Copy tmpdir to "<page review>/.wip/<issue dir>", then move it once the
	// copy succeeded so we can avoid broken dir moves
	// TODO: Copy tmpdir -> config.PDFPageReviewPath/.wip/issuekey

	// Copy the original file(s) into a "-wip" folder, remove the original, and
	// then rename the "-wip" folder
	// TODO: copy to config.MasterPDFBackupPath
}

/*
  def split_valid(self):
    """Finds and splits valid PDFs found in the master path"""
    self.find()
    for pdf_dir in self.pdf_dir_lookup.itervalues():
      tempdir = tempfile.mkdtemp()
      self.process_issue(pdf_dir, tempdir)
      shutil.rmtree(tempdir)

  def find(self):
    self.pdf_dir_lookup = {}
    for fname in utils.find(self.master_path, "*.pdf"):
      path, pdfname = os.path.split(fname)
      if path not in self.pdf_dir_lookup:
        self.pdf_dir_lookup[path] = PDFDir(path, self.master_path, self.namespace, self.out_path, self.backup_path)

      p = self.pdf_dir_lookup[path]
      p.add_file(fname)

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

  def create_master_pdf(self, pdf_dir, master_pdf):
    # Combine pages and/or pre-process PDFs - ghostscript seems to be able to
    # handle some PDFs that crash poppler utils (even as recent as 0.41)
    self.log.debug("Preprocessing with ghostscript")
    files = sorted(pdf_dir.pdfs)
    shell_command = [
      settings.GHOSTSCRIPT, "-sDEVICE=pdfwrite", "-dCompatibilityLevel=1.6",
      "-dPDFSETTINGS=/default", "-dNOPAUSE", "-dQUIET", "-dBATCH", "-dDetectDuplicateImages",
      "-dCompressFonts=true", "-r150", "-sOutputFile=%s" % master_pdf
    ]
    for f in files:
      shell_command.append(f)
    utils.shell(shell_command)

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
    e = pdf_dir.error()
    if e:
      self.log.error("Skipping %s - %s" % (pdf_dir.full_path, e))
      self.log.debug(repr(pdf_dir.__dict__))
      return

    master_pdf = "%s/master.pdf" % tempdir
    self.create_master_pdf(pdf_dir, master_pdf)

    # Split pages to ensure exactly 1 per PDF
    self.log.info("Splitting PDF(s) in '%s'" % pdf_dir.full_path)
    utils.shell(["pdfseparate", master_pdf, "%s/seq-%%d.pdf" % tempdir])

    # Check for an issue with too few pages
    pagecount = len(utils.find(tempdir, "seq-*.pdf"))
    if pagecount < settings.MINIMUM_ISSUE_PAGES:
      self.log.error("Skipping %s - too few PDFs (found %d page(s); need %d)" % (
          pdf_dir.full_path, pagecount, settings.MINIMUM_ISSUE_PAGES))
      return

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
