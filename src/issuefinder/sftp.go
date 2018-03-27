package issuefinder

import (
	"apperr"
	"path/filepath"
	"schema"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

// FindSFTPIssues aggregates all the uploaded born-digital PDFs
func (s *Searcher) FindSFTPIssues(orgCode string) error {
	s.init()

	// First find all titles
	var titlePaths, err = fileutil.FindDirectories(s.Location)
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

// findSFTPIssuesForTitle finds all issues within the given title's path by
// looking for YYYY-MM-DD formatted directories.  The last directory element in
// the path must be an SFTP title name or an LCCN.
func (s *Searcher) findSFTPIssuesForTitlePath(titlePath, orgCode string) error {
	var title = s.findOrCreateFilesystemTitle(titlePath)

	var issuePaths, err = fileutil.FindDirectories(titlePath)
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
			issue.AddError(apperr.Errorf("manually flagged issue"))
			base = base[:len(base)-6]
		}

		var _, err = time.Parse("2006-01-02", base)
		// Invalid issue directory names will have an invalid date, but still need
		// to be visible in the issue queue
		if err != nil {
			issue.AddError(apperr.Errorf("issue folder date format, %q, is invalid", filepath.Base(issuePath)))
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
