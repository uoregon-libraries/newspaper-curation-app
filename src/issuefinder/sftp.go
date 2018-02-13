package issuefinder

import (
	"fmt"
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
		// We don't know the issue (or even if there is an issue object) yet, so we
		// need to aggregate errors.  And we shortcut the aggregation so we don't
		// forget to set the title.
		var errors []*Error
		var addErr = func(e error) { errors = append(errors, s.newError(issuePath, e).SetTitle(title)) }

		// A suffix of "-error" is a manually flagged error; we should keep an eye
		// on these, but their contents can still be valuable
		if strings.HasSuffix(base, "-error") {
			addErr(fmt.Errorf("manually flagged issue"))
			base = base[:len(base)-6]
		}

		var dt, err = time.Parse("2006-01-02", base)
		// Invalid issue directory names will have an invalid date, but still need
		// to be visible in the issue queue
		if err != nil {
			addErr(fmt.Errorf("issue folder date format, %q, is invalid", filepath.Base(issuePath)))
		}

		// Build the issue now that we know we can put together the minimal metadata
		var issue = title.AddIssue(&schema.Issue{
			MARCOrgCode:  orgCode,
			Date:         dt,
			Edition:      1,
			Location:     issuePath,
			WorkflowStep: schema.WSSFTP,
		})
		issue.FindFiles()

		for _, e := range errors {
			e.SetIssue(issue)
		}
		s.Issues = append(s.Issues, issue)
		s.verifyIssueFiles(issue, []string{".pdf"})
	}

	return nil
}
