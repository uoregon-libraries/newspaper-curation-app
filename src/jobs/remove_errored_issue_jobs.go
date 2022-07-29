// remove_errored_issue_jobs.go houses the jobs responsible for taking an issue
// out of NCA entirely. WriteActionLog could be repurposed, and maybe should be
// moved somewhere more generic, but the takeaway here is that for the most
// part you don't use jobs here unless you mean to purge the issue from NCA.

package jobs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/wordutils"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
)

// WriteActionLog is a job that serializes all the actions takes on a given
// issue into simple text for issues that need processing outside of NCA
type WriteActionLog struct {
	*IssueJob
}

// Process iterates over all workflow actions and writes them out to a text
// buffer, which is then written to a file in the issue directory
func (j *WriteActionLog) Process(*config.Config) bool {
	var errPath = filepath.Join(j.DBIssue.Location, "actions.txt")
	var f = fileutil.NewSafeFile(errPath)

	var list = j.DBIssue.AllWorkflowActions()
	for _, a := range list {
		var out = fmt.Sprintf("<%s> %s on %s", a.Author().Login, a.Type().Describe(), a.CreatedAt.Format("on Jan 2, 2006 at 3:04pm"))

		if a.Message != "" {
			var msg = wrapMessage(a.Message)
			out += ":\n\n" + msg
		}

		var _, err = fmt.Fprint(f, out+"\n\n")
		if err != nil {
			j.Logger.Errorf("Unable to write action log to %q: %s", errPath, err)
			f.Cancel()
			return false
		}
	}

	var err = f.Close()
	if err != nil {
		j.Logger.Errorf("Unable to write action log to %q: %s", errPath, err)
		return false
	}

	j.Logger.Infof("Action log written to %q", errPath)
	return true
}

func wrapMessage(msg string) string {
	var full = strings.Replace(msg, "\r\n", "\n", -1)
	full = strings.Replace(full, "\r", "\n", -1)
	var lines = strings.Split(full, "\n")
	for i, line := range lines {
		lines[i] = wrapIndent(line)
	}
	return strings.Join(lines, "\n")
}

func wrapIndent(text string) string {
	var lines = strings.Split(wordutils.Wrap(text, 80), "\n")
	for i, line := range lines {
		lines[i] = "    " + line
	}
	return strings.Join(lines, "\n")
}

// MoveDerivatives tries to get all non-primary content moved from an issue
// directory into a sibling directory so the primary content is isolated and
// easier to re-process if needed
type MoveDerivatives struct {
	*IssueJob
	dest string
}

// Process finds all derivative files, and moves them from the issue location
// to the destination-arg location.  Derivatives, in this process, are defined
// as anything with ".xml" or ".jp2" as its extension.
func (j *MoveDerivatives) Process(*config.Config) bool {
	var src = j.DBIssue.Location
	var dst = j.db.Args[locArg]
	if !fileutil.MustNotExist(dst) {
		j.Logger.Errorf("Destination %q already exists", dst)
		return false
	}
	var err = os.MkdirAll(dst, 0700)
	if err != nil {
		j.Logger.Errorf("Unable to create directory %q: %s", dst, err)
		return false
	}

	var infos []os.FileInfo
	infos, err = ioutil.ReadDir(src)
	if err != nil {
		j.Logger.Errorf("Unable to read source directory %q: %s", src, err)
		return false
	}

	for _, info := range infos {
		var ext = filepath.Ext(info.Name())
		if ext != ".xml" && ext != ".jp2" {
			continue
		}

		var srcFull = filepath.Join(src, info.Name())
		var dstFull = filepath.Join(dst, info.Name())
		err = os.Rename(srcFull, dstFull)
		if err != nil {
			j.Logger.Errorf("Unable to move %q -> %q: %s", srcFull, dstFull, err)
			return false
		}
	}

	return true
}
