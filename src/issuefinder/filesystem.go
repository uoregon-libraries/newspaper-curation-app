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
func (s *Searcher) FindSFTPIssues() error {
	s.init()

	// First find all titles
	var titlePaths, err = fileutil.FindDirectories(s.Location)
	if err != nil {
		return err
	}

	// Find all issues next
	for _, titlePath := range titlePaths {
		err = s.findSFTPIssuesForTitlePath(titlePath)
		if err != nil {
			return err
		}
	}

	return nil
}

// findSFTPIssuesForTitle finds all issues within the given title's path by
// looking for YYYY-MM-DD formatted directories.  As the path is expected to be
// "standard", the last directory element in the path must be an SFTP title
// name or an LCCN.
func (s *Searcher) findSFTPIssuesForTitlePath(titlePath string) error {
	// Make sure we have a legitimate title - we have to check titles by
	// directory and LCCN
	var titleName = filepath.Base(titlePath)
	var title = s.findFilesystemTitle(titleName, titlePath)

	// A missing title is a problem for all standard directory layouts, because
	// these are always in-house issues.  Live batches or old batches on the
	// filesystem wouldn't hit this check.
	//
	// Note that despite not having a valid title we still scan the directory in
	// order to catch other errors and aggregate the unknown titles' issues.
	if title == nil {
		title = &schema.Title{LCCN: titlePath}
		s.addTitle(title)
		s.newError(titlePath, fmt.Errorf("unable to find title %#v in database", titleName))
	}

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
		// Invalid issue directory names can't have an issue, so we can continue
		// without fixing up the errors
		if err != nil {
			addErr(fmt.Errorf("invalid issue directory name: must be formatted YYYY-MM-DD or YYYYMMDD"))
			continue
		}

		var issue = title.AddIssue(&schema.Issue{Date: dt, Edition: 1, Location: issuePath})

		issue.FindFiles()

		for _, e := range errors {
			e.SetIssue(issue)
		}
		s.Issues = append(s.Issues, issue)
		s.verifySFTPIssueFiles(issue)
	}

	return nil
}

// verifySFTPIssueFiles looks for errors in any files within a given issue.
// In our standard layout, the following are considered errors:
// - There are files that aren't regular (symlinks, directories, etc)
// - There are files that aren't pdf
// - The issue directory is empty
func (s *Searcher) verifySFTPIssueFiles(issue *schema.Issue) {
	if len(issue.Files) == 0 {
		s.newError(issue.Location, fmt.Errorf("no issue files found")).SetIssue(issue)
		return
	}

	for _, file := range issue.Files {
		var makeErr = func(format string, args ...interface{}) {
			s.newError(file.Location, fmt.Errorf(format, args...)).SetFile(file)
		}

		if file.IsDir() {
			makeErr("%q is a subdirectory", file.Name)
			continue
		}

		if !file.IsRegular() {
			makeErr("%q is not a regular file", file.Name)
			continue
		}

		var ext = strings.ToLower(filepath.Ext(file.Name))
		if ext != ".pdf" {
			makeErr("%q has an invalid extension", file.Name)
			continue
		}
	}
}
