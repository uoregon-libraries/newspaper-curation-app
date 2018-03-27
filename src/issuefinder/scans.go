package issuefinder

import (
	"apperr"
	"db"
	"os"
	"path/filepath"
	"regexp"
	"schema"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

var pdfFilenameRegex = regexp.MustCompile(`(?i:^[0-9]{4}.pdf)`)
var tiffFilenameRegex = regexp.MustCompile(`(?i:^[0-9]{4}.tiff?)`)

// FindScannedIssues aggregates all the in-house scans waiting for processing
func (s *Searcher) FindScannedIssues() error {
	s.init()

	// First find all MARC org code directories
	var mocPaths, err = fileutil.FindDirectories(s.Location)
	if err != nil {
		return err
	}

	// Any MOCs not in the app are errors and we don't even try to handle them;
	// this should be a pretty rare occurrence for us
	var validMOCPaths []string
	for _, mocPath := range mocPaths {
		var mocName = filepath.Base(mocPath)
		if !db.ValidMOC(mocName) {
			s.Errors = append(s.Errors, apperr.Errorf("unable to find MARC Org Code %#v in database", mocName))
			continue
		}

		validMOCPaths = append(validMOCPaths, mocPath)
	}

	// Next, find titles
	for _, mocPath := range validMOCPaths {
		var paths, err = fileutil.FindDirectories(mocPath)
		if err != nil {
			return err
		}

		// Find the issues within this title path
		var moc = filepath.Base(mocPath)
		for _, titlePath := range paths {
			err = s.findScannedIssuesForTitlePath(moc, titlePath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// findScannedIssuesForTitlePath finds all issues within the given title's path
// by looking for YYYY-MM-DD or YYYY-MM-DD_EE formatted directories.  The
// following error conditions are checked and recorded:
// - The last element of the path isn't a valid title name or LCCN
// - The issue directory isn't a valid date or date/edition combo
// - The issue has no TIFFs
// - The PDFs and TIFFs don't match up (same number of PDFs as TIFFs and same filenames)
func (s *Searcher) findScannedIssuesForTitlePath(moc, titlePath string) error {
	var title = s.findOrCreateFilesystemTitle(titlePath)

	var issuePaths, err = fileutil.FindDirectories(titlePath)
	if err != nil {
		return err
	}

	for _, issuePath := range issuePaths {
		var base = filepath.Base(issuePath)

		// Set up the core of the issue data so we can start attaching errors
		var issue = &schema.Issue{
			Location:     issuePath,
			WorkflowStep: schema.WSScan,
			MARCOrgCode:  moc,
		}

		// If we have an edition, split it off and store it, otherwise it's 1
		var edition = 1
		if len(base) == 13 {
			var edStr = base[11:]
			edition, err = strconv.Atoi(edStr)
			if err != nil {
				issue.AddError(apperr.Errorf("invalid issue directory name: non-numeric edition value %q", edStr))
			} else if edition < 1 {
				issue.AddError(apperr.Errorf("invalid issue directory name: edition must be 1 or greater"))
			}
			base = base[:10]
		}

		// Finish the issue metadata and do final validations
		issue.RawDate = base
		issue.Edition = edition
		issue.FindFiles()
		title.AddIssue(issue)

		var _, err = time.Parse("2006-01-02", base)
		if err != nil {
			issue.AddError(apperr.Errorf("issue folder date format, %q, is invalid", filepath.Base(issuePath)))
		}

		// Make sure PDF and TIFF pairs match up properly
		var verifyErr = s.verifyScanIssuePDFTIFFPairs(issuePath)
		if verifyErr != nil {
			issue.AddError(verifyErr)
		}

		s.Issues = append(s.Issues, issue)
		s.verifyIssueFiles(issue, []string{".pdf", ".tif", ".tiff"})
	}

	return nil
}

func (s *Searcher) verifyScanIssuePDFTIFFPairs(path string) apperr.Error {
	var tiffFiles, pdfFiles []string
	var err error

	tiffFiles, err = fileutil.FindIf(path, func(i os.FileInfo) bool {
		return tiffFilenameRegex.MatchString(i.Name())
	})
	if err == nil {
		pdfFiles, err = fileutil.FindIf(path, func(i os.FileInfo) bool {
			return pdfFilenameRegex.MatchString(i.Name())
		})
	}

	if err != nil {
		return apperr.Errorf("unable to scan %q for PDF / TIFF files: %s", path, err)
	}

	if len(tiffFiles) == 0 {
		return apperr.Errorf("no TIFF files in %q", path)
	}

	if len(tiffFiles) != len(pdfFiles) {
		return apperr.Errorf("PDF/TIFF files don't match in %q", path)
	}

	sort.Strings(tiffFiles)
	sort.Strings(pdfFiles)

	for i, pdf := range pdfFiles {
		var tiff = tiffFiles[i]
		var pdfParts = strings.Split(pdf, ".")
		var tiffParts = strings.Split(tiff, ".")
		if pdfParts[0] != tiffParts[0] {
			return apperr.Errorf("PDF/TIFF files don't match (index %d / pdf %q / tiff %q) in %q", i, pdf, tiff, path)
		}
	}

	return nil
}
