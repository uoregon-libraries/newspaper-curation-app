package issuefinder

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/apperr"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

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
		if !models.ValidMOC(mocName) {
			s.Errors.Append(apperr.Errorf("unable to find MARC Org Code %#v in database", mocName))
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
				issue.ErrInvalidFolderName("non-numeric edition value")
			} else if edition < 1 {
				issue.ErrInvalidFolderName("edition must be 1 or greater")
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
			issue.ErrInvalidFolderName("bad date format")
		}

		// Make sure PDF and TIFF pairs match up properly
		s.verifyScanIssuePDFTIFFPairs(issue)

		s.Issues = append(s.Issues, issue)
		s.verifyIssueFiles(issue, []string{".pdf", ".tif", ".tiff"})
	}

	return nil
}

func (s *Searcher) verifyScanIssuePDFTIFFPairs(issue *schema.Issue) {
	var tiffFiles, pdfFiles []string

	var infos, err = fileutil.ReaddirSortedNumeric(issue.Location)
	if err != nil {
		issue.ErrReadFailure(err)
		return
	}

	for _, info := range infos {
		switch strings.ToLower(filepath.Ext(info.Name())) {
		case ".pdf":
			pdfFiles = append(pdfFiles, info.Name())
		case ".tif", ".tiff":
			tiffFiles = append(tiffFiles, info.Name())
		}
	}

	if len(tiffFiles) == 0 {
		issue.ErrFolderContents("one or more TIFF files must be present")
		return
	}

	if len(tiffFiles) != len(pdfFiles) {
		issue.ErrFolderContents("PDF/TIFF files don't match")
		return
	}

	sort.Strings(tiffFiles)
	sort.Strings(pdfFiles)

	for i, pdf := range pdfFiles {
		var tiff = tiffFiles[i]
		var pdfParts = strings.Split(pdf, ".")
		var tiffParts = strings.Split(tiff, ".")
		if pdfParts[0] != tiffParts[0] {
			issue.ErrFolderContents(fmt.Sprintf("PDF/TIFF files don't match (index %d / pdf %q / tiff %q)", i, pdf, tiff))
			return
		}
	}
}
