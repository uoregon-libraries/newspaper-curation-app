package jobs

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/uoregon-libraries/gopkg/fileutil"
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
	var list = j.DBIssue.WorkflowActions()
	var output = new(bytes.Buffer)
	for _, a := range list {
		fmt.Fprintln(output, a.ActionType)
	}

	// TODO: Write to file
	j.Logger.Errorf("WriteActionLog is not yet implemented")

	return false
}

// MoveDerivatives tries to get all non-primary content moved from an issue
// directory into a sibling directory so the primary content is isolated and
// easier to re-process if needed
type MoveDerivatives struct {
	*IssueJob
	dest string
}

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
