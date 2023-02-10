package issuefinder

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
)

// FindSFTPIssues aggregates all the uploaded born-digital PDFs
func (s *Searcher) FindSFTPIssues(orgCode string) error {
	var err = s.init()
	if err != nil {
		return err
	}

	// First find all titles
	var titlePaths []string
	titlePaths, err = fileutil.FindDirectories(s.Location)
	if err != nil {
		return err
	}

	// Find all issues next
	for _, titlePath := range titlePaths {
		err = s.findSFTPIssuesForTitlePath(titlePath, orgCode)
		if err != nil {
			return err
		}
	}

	return nil
}

func findPDFDirs(root string) (results []string, err error) {
	var infos []os.FileInfo
	infos, err = fileutil.ReaddirSortedNumeric(root)
	if err != nil {
		return nil, err
	}

	var hasPDF bool
	for _, info := range infos {
		if info.IsDir() {
			var res2, err2 = findPDFDirs(filepath.Join(root, info.Name()))
			if err2 != nil {
				return nil, err2
			}
			results = append(results, res2...)
		}

		var ext = strings.ToLower(filepath.Ext(info.Name()))
		if info.Mode().IsRegular() && ext == ".pdf" {
			hasPDF = true
		}
	}

	if hasPDF {
		results = append(results, root)
	}
	return results, err
}

// findSFTPIssuesForTitle finds all issues within the given title's path by
// looking for YYYY-MM-DD formatted directories.  The last directory element in
// the path must be an SFTP title name or an LCCN.
func (s *Searcher) findSFTPIssuesForTitlePath(titlePath, orgCode string) error {
	var title = s.findOrCreateFilesystemTitle(titlePath)

	// Find all dirs (up to a depth of 3) that have at least one *.pdf in them.
	// Then turn those into issues as best we can; this may cause us to lose
	// visibility of busted non-PDF uploads, but it removes the previous problem
	// where issues uploaded "too deep" caused NCA to show errors that made no
	// sense to the end user.
	var issuePaths, err = findPDFDirs(titlePath)
	if err != nil {
		return err
	}

	for _, issuePath := range issuePaths {
		var base = filepath.Base(issuePath)

		// Set up the core of the issue data so we can start attaching errors
		var issue = &schema.Issue{
			MARCOrgCode:  orgCode,
			Location:     issuePath,
			WorkflowStep: schema.WSSFTP,
		}

		// A suffix of "-error" is a manually flagged error; we should keep an eye
		// on these, but their contents can still be valuable
		if strings.HasSuffix(base, "-error") {
			issue.ErrInvalidFolderName("manually flagged issue")
			base = base[:len(base)-6]
		}

		var _, err = time.Parse("2006-01-02", base)
		// Invalid issue directory names will have an invalid date, but still need
		// to be visible in the issue queue
		if err != nil {
			issue.ErrInvalidFolderName("bad date format")
		}

		// Finish the issue metadata and do final validations
		issue.RawDate = base
		issue.Edition = 1
		issue.FindFiles()
		title.AddIssue(issue)

		s.Issues = append(s.Issues, issue)
		s.verifyIssueFiles(issue, []string{".pdf"})
	}

	return nil
}
