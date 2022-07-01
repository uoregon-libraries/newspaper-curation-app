package schema

import (
	"io/ioutil"
	"os"
	"time"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

// LastModified tells us when *any* change happened in an issue's folder.  This
// will return a meaningless value on live issues.
func (i *Issue) LastModified() time.Time {
	if i.WorkflowStep == WSInProduction {
		return time.Time{}
	}

	var info, err = os.Stat(i.Location)
	if err != nil {
		logger.Warnf("Unable to stat %q: %s", i.Location, err)
		return time.Now()
	}
	var modified = info.ModTime()

	var files []os.FileInfo
	files, err = ioutil.ReadDir(i.Location)
	if err != nil {
		logger.Warnf("Unable to read dir %q: %s", i.Location, err)
		return time.Now()
	}

	for _, file := range files {
		var mod = file.ModTime()
		if modified.Before(mod) {
			modified = mod
		}
	}

	return modified
}
