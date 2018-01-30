package issuefinder

import (
	"db"
	"fmt"
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
			s.newError(mocPath, fmt.Errorf("unable to find MARC Org Code %#v in database", mocName))
			continue
		}

		validMOCPaths = append(validMOCPaths, mocPath)
	}

	// Next, find titles
	var titlePaths []string
	for _, mocPath := range validMOCPaths {
		var paths, err = fileutil.FindDirectories(mocPath)
		if err != nil {
			return err
		}
		titlePaths = append(titlePaths, paths...)
	}

	// Finally, find the issues
	for _, titlePath := range titlePaths {
		err = s.findScannedIssuesForTitlePath(titlePath)
		if err != nil {
			return err
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
// - Any PDF has an image with an unexpected DPI (using our gopkg/pdf lib)
func (s *Searcher) findScannedIssuesForTitlePath(titlePath string) error {
	var title = s.findOrCreateFilesystemTitle(titlePath)

	var issuePaths, err = fileutil.FindDirectories(titlePath)
	if err != nil {
		return err
	}

	for _, issuePath := range issuePaths {
		var base = filepath.Base(issuePath)
		// We don't know the issue (or even if there is an issue object) yet, so we
		// need to aggregate errors.  And we shortcut the aggregation so we don't
		// forget to set the title.
		var errors []*Error
		var addErr = func(e error) { errors = append(errors, s.newError(issuePath, e).SetTitle(title)) }

		// If we have an edition, split it off and store it, otherwise it's 1
		var edition = 1
		if len(base) == 13 {
			var edStr = base[11:]
			edition, err = strconv.Atoi(edStr)
			if err != nil {
				addErr(fmt.Errorf("invalid issue directory name: non-numeric edition value %q", edStr))
				continue
			}
			if edition < 1 {
				addErr(fmt.Errorf("invalid issue directory name: edition must be 1 or greater"))
				continue
			}
			base = base[:10]
		}

		var dt, err = time.Parse("2006-01-02", base)
		// Invalid issue directory names can't have an issue, so we can continue
		// without fixing up the errors
		if err != nil {
			addErr(fmt.Errorf("invalid issue directory name: must be formatted YYYY-MM-DD or YYYY-MM-DD_EE"))
			continue
		}

		// Build the issue now that we know we can put together the minimal metadata
		var issue = title.AddIssue(&schema.Issue{Date: dt, Edition: edition, Location: issuePath})
		issue.FindFiles()

		// Make sure PDF and TIFF pairs match up properly
		err = s.verifyScanIssuePDFTIFFPairs(issuePath)
		if err != nil {
			addErr(err)
			continue
		}

		for _, e := range errors {
			e.SetIssue(issue)
		}
		s.Issues = append(s.Issues, issue)
		s.verifyIssueFiles(issue, []string{".pdf", ".tif", ".tiff"})
	}

	return nil
}

func (s *Searcher) verifyScanIssuePDFTIFFPairs(path string) error {
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
		return fmt.Errorf("unable to scan %q for PDF / TIFF files: %s", path, err)
	}

	if len(tiffFiles) == 0 {
		return fmt.Errorf("no TIFF files in %q", path)
	}

	if len(tiffFiles) != len(pdfFiles) {
		return fmt.Errorf("PDF/TIFF files don't match in %q", path)
	}

	sort.Strings(tiffFiles)
	sort.Strings(pdfFiles)

	for i, pdf := range pdfFiles {
		var tiff = tiffFiles[i]
		var pdfParts = strings.Split(pdf, ".")
		var tiffParts = strings.Split(tiff, ".")
		if pdfParts[0] != tiffParts[0] {
			return fmt.Errorf("PDF/TIFF files don't match (index %d / pdf %q / tiff %q) in %q", i, pdf, tiff, path)
		}
	}

	return nil
}
