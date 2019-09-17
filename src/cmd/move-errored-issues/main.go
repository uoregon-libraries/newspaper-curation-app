package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

// Command-line options
type _opts struct {
	cli.BaseOptions
	Dest string `long:"destination" description:"location to move issues" required:"true"`
}

var opts _opts
var titles db.TitleList

func getOpts() *config.Config {
	var c = cli.New(&opts)
	c.AppendUsage("Finds all batches which were flagged as having errors, " +
		"moves them out of the workflow location to the given --destination, " +
		"and updates the database so the issues are no longer seen by NCA.")
	var conf = c.GetConf()
	var err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}

	if !fileutil.IsDir(opts.Dest) {
		c.UsageFail(fmt.Sprintf("Destination %q is invalid", opts.Dest))
	}

	return conf
}

func main() {
	getOpts()
	logger.Infof("Finding errored issues to move")
	var issues, err = db.FindIssuesWithErrors()
	if err != nil {
		logger.Fatalf("Unable to query the database for issues: %s", err)
	}

	for _, issue := range issues {
		if !moveIssue(issue, opts.Dest) {
			break
		}
	}
}

// moveIssue attempts to move an issue from its current location to a new
// location.  The return value tells us whether the move was successful enough
// to continue moving other issues.
func moveIssue(issue *db.Issue, dest string) (ok bool) {
	logger.Infof("Attempting to move issue %d (location: %q)", issue.ID, issue.Location)
	// We set this to true when we want the operation on this issue to continue
	// (to get the data as "good" as possible), but still report "not okay"
	var failure = false

	var dateEdition = fmt.Sprintf("%s_%02d", issue.Date, issue.Edition)
	var contentDest = filepath.Join(dest, "content", issue.LCCN, dateEdition)
	var derivDest = filepath.Join(dest, "derivatives", issue.LCCN, dateEdition)
	var masterDest = filepath.Join(contentDest, "master")

	logger.Debugf("Moving content files from %q to %q", issue.Location, contentDest)
	var err = moveDir(issue.Location, contentDest)
	if err != nil {
		var merr, ok = err.(*moveError)
		if !ok || !merr.didCopy {
			logger.Errorf("Unable to copy issue from %q to %q: %s", issue.Location, contentDest, err)
			return false
		}

		logger.Errorf("Unable to remove issue in %q: %s (you must remove this manually!)", issue.Location, err)
		failure = true
	}

	logger.Debugf("Moving all other files from %q to %q", issue.Location, derivDest)
	err = moveDerivativeFiles(contentDest, derivDest)
	if err != nil {
		logger.Errorf("Unable to move derivatives from %q to %q: %s", contentDest, derivDest, err)
		failure = true
	}

	// Drop a file into the copied directory with the error notes
	var errFile = filepath.Join(contentDest, "error.txt")
	logger.Debugf("Writing errors to %q", errFile)
	err = ioutil.WriteFile(errFile, []byte(issue.Error), 0660)
	if err != nil {
		logger.Errorf("Unable to create error.txt file %q: %s", errFile, err)
		failure = true
	}

	// Now we want to move the master files (if they exist)
	if issue.MasterBackupLocation != "" {
		logger.Debugf("Moving masters from %q to %q", issue.MasterBackupLocation, masterDest)
		err = moveDir(issue.MasterBackupLocation, masterDest)
		// Errors while moving the master are very annoying, because the issue's
		// files are already copied.  We just report the error and move on....
		if err != nil {
			logger.Errorf("Unable to move master backup from %q to %q: %s", issue.MasterBackupLocation, masterDest, err)
			failure = true
		}
	}

	// Update the issue so we never deal with it again (but we preserve pretty
	// much all relevant data so we can refer to it if necessary).
	issue.Location = ""
	issue.Ignored = true
	logger.Debugf("Updating issue metadata in database")
	err = issue.Save()

	// If we couldn't save to the database, we're very unhappy.  This is
	// possibly the worst scenario, because it's not all that unlikely compared
	// to other post-copy failures.  The data has been copied and the source
	// removed, so we can't really back out... but we also can't update the
	// issue in the database.  Response?  Somebody has to manually fix the
	// database :-/
	if err != nil {
		logger.Errorf("Unable to update issue %d in the database: %s (you must fix this manually!)", issue.ID, err)
		return false
	}

	logger.Debugf("Done")
	return !failure
}

// moveError lets us wrap an error with extra information so we can tell if the
// move failed after the copy was successful.  With this detail we know if the
// problem is critical and will require re-copying entirely, or if the problem
// just means somebody needs to clean up leftover files.
type moveError struct {
	error
	didCopy bool
}

// moveDir runs the copy / remove logic, returning an error where applicable
func moveDir(src, dst string) error {
	os.MkdirAll(filepath.Dir(dst), 0700)
	var err = fileutil.CopyDirectory(src, dst)
	if err != nil {
		return &moveError{err, false}
	}

	err = os.RemoveAll(src)
	if err != nil {
		return &moveError{err, true}
	}

	return nil
}

// moveDerivativeFiles tries to get all non-primary content moved to the
// destination dir, creating it if necessary
func moveDerivativeFiles(src, dst string) error {
	if !fileutil.MustNotExist(dst) {
		return fmt.Errorf("destination %q already exists", dst)
	}
	var err = os.MkdirAll(dst, 0700)
	if err != nil {
		return fmt.Errorf("unable to create directory %q: %s", dst, err)
	}

	var infos []os.FileInfo
	infos, err = fileutil.Readdir(src)
	if err != nil {
		return fmt.Errorf("unable to read source directory %q: %s", src, err)
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
			return err
		}
	}

	return nil
}
