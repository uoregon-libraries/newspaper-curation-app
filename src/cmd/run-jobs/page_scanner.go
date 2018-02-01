package main

import (
	"config"
	"db"
	"jobs"
	"os"
	"path/filepath"
	"regexp"
	"schema"
	"sort"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/pdf"
)

var pdfFilenameRegex = regexp.MustCompile(`(?i:^[0-9]{4}.pdf)`)
var tiffFilenameRegex = regexp.MustCompile(`(?i:^[0-9]{4}.tiff?)`)

func scanPageReviewIssues(c *config.Config) {
	var list, err = db.FindIssuesInPageReview()
	if err != nil {
		logger.Errorf("Unable to query issues in page review: %s", err)
		return
	}

	for _, dbIssue := range list {
		if issuePagesReady(dbIssue.Location, time.Hour, pdfFilenameRegex) {
			queueIssueForDerivatives(dbIssue)
		}
	}
}

// scanScannedIssues is a terrible name for the very important process of
// looking for in-house scanned issues that are valid and ready for processing
func scanScannedIssues(c *config.Config) {
	var mocDirs = getScannedMOCDirList(c.MasterScanUploadPath)
	var lccnDirs []string
	for _, mocDir := range mocDirs {
		lccnDirs = append(lccnDirs, getLCCNDirs(mocDir)...)
	}
	var dbIssues []*db.Issue
	for _, lccnDir := range lccnDirs {
		dbIssues = append(dbIssues, makeScannedDBIssuesFromLCCNDir(lccnDir)...)
	}

	for _, dbIssue := range dbIssues {
		// Make sure generic "page ready" logic is good
		if !issuePagesReady(dbIssue.Location, time.Hour*24, pdfFilenameRegex, tiffFilenameRegex) {
			continue
		}

		// Make sure we have exactly the same TIFF and PDF files; we check this in
		// the derivative processor, but that's more of a backup validation; it's
		// best to catch errors here so they're easy to fix
		if !validScanFiles(dbIssue.Location) {
			continue
		}

		// Make sure the PDFs' images are at the right DPI
		if !validScanPDFDPI(dbIssue.Location, c.ScannedPDFDPI) {
			continue
		}

		queueIssueForDerivatives(dbIssue)
	}
}

// queueIssueForDerivatives first renames the directory so no more
// modifications are likely to take place, then queues the directory for being
// moved to the workflow space
func queueIssueForDerivatives(dbIssue *db.Issue) {
	var oldDir = dbIssue.Location
	var newDir = filepath.Join(filepath.Dir(oldDir), ".notouchie-"+filepath.Base(oldDir))
	logger.Infof("Renaming %q to %q to prepare for derivative processing", oldDir, newDir)
	var err = os.Rename(oldDir, newDir)
	if err != nil {
		logger.Errorf("Unable to rename %q for derivative processing: %s", oldDir, err)
		return
	}
	dbIssue.Location = newDir
	dbIssue.WorkflowStep = schema.WSAwaitingProcessing
	err = dbIssue.Save()
	if err != nil {
		logger.Criticalf("Unable to update db Issue (location and awaiting page review status): %s", err)
		return
	}

	// Queue up move to workflow dir
	jobs.QueueMoveIssueForDerivatives(dbIssue, newDir)
}

func getScannedMOCDirList(path string) []string {
	var infos, err = fileutil.ReaddirSorted(path)
	if err != nil {
		logger.Errorf("Unable to read scan directory %q: %s", path, err)
		return nil
	}

	var mocDirs []string
	for _, info := range infos {
		// We silently skip top-level files, as there seem to often be things like
		// scan log files generated automatically
		if !info.IsDir() {
			continue
		}
		var code = filepath.Base(info.Name())

		// We shouldn't have directories that aren't in the system
		if !db.ValidMOC(code) {
			logger.Errorf("Invalid MARC Org Code directory: %q", info.Name())
			continue
		}

		mocDirs = append(mocDirs, filepath.Join(path, info.Name()))
	}

	return mocDirs
}

// getLCCNDirs scans the given path for valid LCCN dirs
func getLCCNDirs(path string) []string {
	var infos, err = fileutil.ReaddirSorted(path)
	if err != nil {
		logger.Errorf("Unable to scan %q for LCCN dirs: %s", path, err)
		return nil
	}

	var lccnDirs []string
	for _, info := range infos {
		// We skip top-level files with a warning
		if !info.IsDir() {
			logger.Warnf("Unexpected file found in LCCN dir scan: %q", info.Name())
			continue
		}
		var lccn = filepath.Base(info.Name())

		// We shouldn't have LCCN directories that aren't in the system
		if db.LookupTitle(lccn) == nil {
			logger.Errorf("Invalid LCCN directory: %q", info.Name())
			continue
		}

		lccnDirs = append(lccnDirs, filepath.Join(path, info.Name()))
	}

	return lccnDirs
}

func makeScannedDBIssuesFromLCCNDir(path string) []*db.Issue {
	var infos, err = fileutil.ReaddirSorted(path)
	if err != nil {
		logger.Errorf("Unable to read directory %q: %s", path, err)
		return nil
	}

	var dbIssues []*db.Issue
	for _, info := range infos {
		if !info.IsDir() {
			// We don't abort, but this situation really shouldn't be happening
			logger.Errorf("Unexpected file found in LCCN dir %q while scanning for issues", info.Name())
			continue
		}

		// Ignore scan dirs already prepped
		if strings.HasPrefix(info.Name(), ".notouchie-") {
			continue
		}

		var issueDir = filepath.Join(path, info.Name())
		var dbIssue, err = db.NewIssueFromScanDir(issueDir)
		if err != nil {
			logger.Errorf("Unable to make DB Issue for %q: %s", issueDir, err)
			continue
		}

		dbIssues = append(dbIssues, dbIssue)
	}

	return dbIssues
}

// validScanFiles ensures the PDF and TIFF files match
func validScanFiles(path string) bool {
	var dirs, tiffFiles, pdfFiles []string
	var err error

	dirs, err = fileutil.FindDirectories(path)
	if len(dirs) > 0 {
		logger.Errorf("Found one or more subdirectories in %q", path)
		return false
	}

	tiffFiles, err = fileutil.FindIf(path, func(i os.FileInfo) bool {
		return tiffFilenameRegex.MatchString(i.Name())
	})
	if err == nil {
		pdfFiles, err = fileutil.FindIf(path, func(i os.FileInfo) bool {
			return pdfFilenameRegex.MatchString(i.Name())
		})
	}

	if err != nil {
		logger.Errorf("Unable to scan %q for PDF / TIFF files: %s", path, err)
		return false
	}

	if len(tiffFiles) == 0 {
		logger.Errorf("There are no TIFF files in %q", path)
		return false
	}

	if len(tiffFiles) != len(pdfFiles) {
		logger.Errorf("PDF/TIFF files don't match in %q", path)
		return false
	}

	sort.Strings(tiffFiles)
	sort.Strings(pdfFiles)

	for i, pdf := range pdfFiles {
		var tiff = tiffFiles[i]
		var pdfParts = strings.Split(pdf, ".")
		var tiffParts = strings.Split(tiff, ".")
		if pdfParts[0] != tiffParts[0] {
			logger.Errorf("PDF/TIFF files don't match (index %d / pdf %q / tiff %q) in %q", i, pdf, tiff, path)
			return false
		}
	}

	return true
}

// validScanPDFDPI returns true if all the images in all PDFs are within a
// valid DPI range
func validScanPDFDPI(path string, expectedDPI int) bool {
	var maxDPI = float64(expectedDPI) * 1.15

	var pdfFiles, err = fileutil.FindIf(path, func(i os.FileInfo) bool {
		return pdfFilenameRegex.MatchString(i.Name())
	})
	if err != nil {
		logger.Errorf("Unable to find PDF files in %q: %s", path, err)
	}

	for _, filename := range pdfFiles {
		var dpis = pdf.ImageDPIs(filename)
		if len(dpis) == 0 {
			logger.Errorf("%q has no images or is invalid", filename)
			return false
		}

		for _, dpi := range dpis {
			if dpi.X > maxDPI || dpi.Y > maxDPI {
				logger.Errorf("%q has an image with a bad DPI (%g x %g)", filename, dpi.X, dpi.Y)
				return false
			}
		}
	}

	return true
}