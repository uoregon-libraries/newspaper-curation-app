package issuefinder

import (
	"fmt"
	"path/filepath"
	"schema"
	"strings"
)

// verifyIssueFiles looks for errors in any files within a given issue.  With
// SFTP or scanned issues, the following are considered errors:
// - The issue directory is empty
// - There are files that aren't regular (symlinks, directories, etc)
// - There are files which aren't using on of a strict list of file extensions
//   (hidden files are ignored to avoid annoyances when bridge, for instance,
//   drops off its various metadata files)
func (s *Searcher) verifyIssueFiles(issue *schema.Issue, allowedExtensions []string) {
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

		if file.Name[0] == '.' {
			continue
		}

		var fileExt = strings.ToLower(filepath.Ext(file.Name))
		var match = false
		for _, ext := range allowedExtensions {
			if fileExt == ext {
				match = true
				break
			}
		}
		if !match {
			makeErr("%q has an invalid extension", file.Name)
			continue
		}
	}
}
