package jobs

import (
	"config"

	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"shell"
	"strconv"

	"github.com/uoregon-libraries/gopkg/fileutil"
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
	GhostScript    string // The path to gs for combining the fake master PDF
	MinPages       int    // Number of pages below which we refuse to process
}

// Process combines, splits, and then renames files so they're sequential in a
// "best guess" order.  Files are then put into place for manual processors to
// reorder if necessary, remove duped pages, etc.
func (ps *PageSplit) Process(config *config.Config) bool {
	ps.Logger.Debugf("Processing issue id %d (%q)", ps.DBIssue.ID, ps.Issue.Key())
	if !ps.makeTempFiles() {
		return false
	}
	defer ps.removeTempFiles()

	ps.WIPDir = filepath.Join(config.WorkflowPath, ps.IssueJob.WIPDir())
	ps.MasterBackup = filepath.Join(config.MasterPDFBackupPath, ps.Subdir())

	if !fileutil.MustNotExist(ps.WIPDir) {
		ps.Logger.Errorf("WIP dir %q already exists", ps.WIPDir)
		return false
	}
	if !fileutil.MustNotExist(ps.MasterBackup) {
		ps.Logger.Errorf("Master backup dir %q already exists", ps.MasterBackup)
		return false
	}

	ps.GhostScript = config.GhostScript
	ps.MinPages = config.MinimumIssuePages
	return ps.process()
}

func (ps *PageSplit) makeTempFiles() (ok bool) {
	var err error
	ps.FakeMasterFile, err = fileutil.TempNamedFile("", "splitter-master-", ".pdf")
	if err != nil {
		ps.Logger.Errorf("Unable to create temp file for combining PDFs: %s", err)
		return false
	}

	ps.TempDir, err = ioutil.TempDir("", "splitter-pages-")
	if err != nil {
		ps.Logger.Errorf("Unable to create temp dir for issue processing: %s", err)
		return false
	}

	return true
}

func (ps *PageSplit) removeTempFiles() {
	var err = os.Remove(ps.FakeMasterFile)
	if err != nil {
		ps.Logger.Warnf("Unable to remove temp file %q: %s", ps.FakeMasterFile, err)
	}
	err = os.RemoveAll(ps.TempDir)
	if err != nil {
		ps.Logger.Warnf("Unable to remove temp dir %q: %s", ps.TempDir, err)
	}
}

func (ps *PageSplit) process() (ok bool) {
	ps.updateWorkflowCB = ps.updateIssueWorkflow
	return RunWhileTrue(
		ps.createMasterPDF,
		ps.splitPages,
		ps.fixPageNames,
		ps.convertToPDFA,
		ps.backupOriginals,
		ps.moveIssue,
	)
}

// createMasterPDF combines pages and pre-processes PDFs - ghostscript seems to
// be able to handle some PDFs that crash poppler utils (even as recent as 0.41)
func (ps *PageSplit) createMasterPDF() (ok bool) {
	ps.Logger.Debugf("Preprocessing with ghostscript")

	var fileinfos, err = fileutil.ReaddirSorted(ps.Location)
	if err != nil {
		ps.Logger.Errorf("Unable to list files in %q: %s", ps.Location, err)
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
	return shell.ExecSubgroup(ps.GhostScript, ps.Logger, args...)
}

// splitPages ensures we end up with exactly one PDF per page
func (ps *PageSplit) splitPages() (ok bool) {
	ps.Logger.Infof("Splitting PDF(s)")
	return shell.ExecSubgroup("pdfseparate", ps.Logger, ps.FakeMasterFile, filepath.Join(ps.TempDir, "seq-%d.pdf"))
}

// fixPageNames converts sequenced PDFs to have 4-digit page numbers
func (ps *PageSplit) fixPageNames() (ok bool) {
	ps.Logger.Infof("Renaming pages so they're sortable")
	var fileinfos, err = fileutil.ReaddirSorted(ps.TempDir)
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
	var fileinfos, err = fileutil.ReaddirSorted(ps.TempDir)
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

// backupOriginals stores the original uploads in the master backup location
func (ps *PageSplit) backupOriginals() (ok bool) {
	var masterParent = filepath.Dir(ps.MasterBackup)
	var err = os.MkdirAll(masterParent, 0700)
	if err != nil {
		ps.Logger.Criticalf("Unable to create master backup parent %q: %s", masterParent, err)
		return false
	}

	err = fileutil.CopyDirectory(ps.Location, ps.MasterBackup)
	if err != nil {
		ps.Logger.Criticalf("Unable to copy master file(s) from %q to %q: %s", ps.Location, ps.MasterBackup, err)
		return false
	}

	err = os.RemoveAll(ps.Location)
	if err != nil {
		ps.Logger.Criticalf("Unable to remove original files after making master backup: %s", err)
		return false
	}

	return true
}

// moveIssue moves the processed files into the recently-removed workflow
// location.  Failures here are very not-ideal as we've already successfully
// destroyed the original files.  They still exist in the master backup
// location, but that's not super easy to automatically fix.
func (ps *PageSplit) moveIssue() (ok bool) {
	var err = fileutil.CopyDirectory(ps.TempDir, ps.WIPDir)
	if err != nil {
		ps.Logger.Criticalf("Unable to move temporary directory %q to %q: %s", ps.TempDir, ps.WIPDir, err)
		return false
	}
	err = os.Rename(ps.WIPDir, ps.Location)
	if err != nil {
		ps.Logger.Criticalf("Unable to move WIP directory %q to %q: %s", ps.WIPDir, ps.Location, err)
		return false
	}
	return true
}

// updateIssueWorkflow sets the Issue's location and flips the "awaiting manual
// ordering" flag so we can track the issue with our "move manually ordered
// issues" scanner
func (ps *PageSplit) updateIssueWorkflow() {
	ps.DBIssue.MasterBackupLocation = ps.MasterBackup
}
