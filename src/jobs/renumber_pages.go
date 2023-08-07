package jobs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
)

// RenumberPages is a job which finds all the PDF and TIFF files and renumbers
// them so that we can always expect 0001.pdf, 0002.pdf, etc., properly
// matching the NDNP spec.
type RenumberPages struct {
	*IssueJob
}

// splitExt uses the same logic filepath.Ext uses to get a file's extension,
// but gives the prefix in addition to the extension
func splitExt(fname string) (prefix, ext string) {
	for i := len(fname) - 1; i >= 0 && !os.IsPathSeparator(fname[i]); i-- {
		if fname[i] == '.' {
			return fname[:i], fname[i:]
		}
	}
	return fname, ""
}

// Process renumbers all page filenames
func (j *RenumberPages) Process(*config.Config) ProcessResponse {
	j.Logger.Debugf("Starting renumber-pages job for issue id %d", j.DBIssue.ID)

	j.Issue.FindFiles()
	if len(j.Issue.Files) == 0 {
		j.Logger.Errorf("No files found")
		return PRFailure
	}

	// First gather all the prefixes so we can map, for instance, "gray0356" to
	// "0001", "gray0357" to "0002", etc.
	var prefixMap = make(map[string]int)
	var num = 1
	for _, f := range j.Issue.Files {
		var pre, ext = splitExt(f.Name)
		var lExt = strings.ToLower(ext)
		if lExt != ".pdf" && lExt != ".tif" && lExt != ".tiff" {
			j.Logger.Debugf("Ignoring file %q: not a source PDF / TIFF", f.Name)
			continue
		}

		j.Logger.Debugf("Examining file %q", f.Name)
		if prefixMap[pre] == 0 {
			j.Logger.Debugf("Prefix %q mapped to file number %d", pre, num)
			prefixMap[pre] = num
			num++
		}

		var newName = fmt.Sprintf("%04d%s", prefixMap[pre], lExt[:4])
		var newLoc = filepath.Join(j.Issue.Location, newName)
		j.Logger.Debugf("Renaming %q to %q", f.Location, newLoc)
		var err = os.Rename(f.Location, newLoc)
		if err != nil {
			j.Logger.Errorf("Error renaming %q to %q: %s", f.Location, newLoc, err)
			return PRFailure
		}
	}

	j.Issue.FindFiles()
	j.Logger.Debugf("Successfully renumbered pages for issue %d", j.DBIssue.ID)
	return PRSuccess
}
