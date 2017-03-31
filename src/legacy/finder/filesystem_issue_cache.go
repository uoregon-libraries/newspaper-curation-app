package main

import (
	"fileutil"
	"log"
	"path/filepath"
	"strings"
	"time"
)

// cacheAllFilesystemIssues calls all the individual cache functions for the
// myriad of ways we store issue information in the various locations
func cacheAllFilesystemIssues() {
	var err error

	err = cacheSFTPIssues()
	if err != nil {
		log.Fatalf("Error trying to cache SFTPed issues: %s", err)
	}
	err = cacheStandardIssues()
	if err != nil {
		log.Fatalf("Error trying to cache standard filesystem issues: %s", err)
	}
}

// cacheSFTPIssues is just barely its own special case because unlike the
// standard structure, there is no "topdir" element in the paths
func cacheSFTPIssues() error {
	// First find all titles
	var titlePaths, err = fileutil.FindDirectories(Conf.MasterPDFUploadPath)
	if err != nil {
		return err
	}

	// Find all issues next
	for _, titlePath := range titlePaths {
		err = cacheStandardIssuesForTitle(titlePath, false)
		if err != nil {
			return err
		}
	}

	return nil
}

// cacheStandardIssues deals with all the various locations for issues which
// are not in a batch directory structure.  This doesn't mean they haven't been
// batched, just that the directory uses the somewhat consistent pdf-to-chronam
// structure `topdir/sftpnameOrLCCN/yyyy-mm-dd/`
func cacheStandardIssues() error {
	var locs = []string{
		Conf.MasterPDFBackupPath,
		Conf.PDFPageReviewPath,
		Conf.PDFPagesAwaitingMetadataReview,
		Conf.PDFIssuesAwaitingDerivatives,
		Conf.ScansAwaitingDerivatives,
		Conf.PDFPageBackupPath,
		Conf.PDFPageSourcePath,
	}

	for _, loc := range locs {
		var err = cacheStandardIssuesFromPath(loc)
		if err != nil {
			return err
		}
	}

	return nil
}

// cacheStandardIssuesFromPath does the work of finding and returning all issue
// information within a given path with the assumption that the path conforms
// to `topdir/sftpnameOrLCCN/yyyy-mm-dd/`
func cacheStandardIssuesFromPath(path string) error {
	// First find all topdirs
	var topdirs, err = fileutil.FindDirectories(path)
	if err != nil {
		return err
	}

	// Next, find titles
	var titlePaths []string
	for _, p := range topdirs {
		var paths, err = fileutil.FindDirectories(p)
		if err != nil {
			return err
		}

		titlePaths = append(titlePaths, paths...)
	}

	// Finally, find issues
	for _, titlePath := range titlePaths {
		err = cacheStandardIssuesForTitle(titlePath, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// cacheStandardIssuesForTitle finds all issues within the given title's path
// by looking for YYYY-MM-DD formatted directories.  The path is expected to be
// "standard", so the last directory element in the path must be an SFTP title
// name or an LCCN.
func cacheStandardIssuesForTitle(path string, allowEdition bool) error {
	// Make sure we have a legitimate title - we have to check both the SFTP
	// and LCCN lookups
	var titleName = filepath.Base(path)
	var title = titlesBySFTPDir[titleName]
	if title == nil {
		title = titlesByLCCN[titleName]
	}

	// Not having a title is a problem, but not a reason to fail the whole
	// process, so we log an error while letting the caller continue
	if title == nil {
		log.Printf("ERROR: Invalid title detected in %#v: %s", path, titleName)
		return nil
	}

	var issuePaths, err = fileutil.FindDirectories(path)
	if err != nil {
		return err
	}

	for _, issuePath := range issuePaths {
		var base = filepath.Base(issuePath)
		// To avoid excessive errors, we can skip anything ending in "-error", as
		// that's currently one way we flag problems
		if strings.HasSuffix(base, "-error") {
			continue
		}

		// Oh, and sometimes it's okay to have _\d\d in the path.  Technically this
		// isn't okay for the SFTP uploads, though, so it's an arg, not an
		// always-on check.
		if allowEdition && len(base) >= 13 && base[10] == '_' {
			base = base[:10] + base[13:]
		}

		// And of course we have to remove our wonderful path hack that was built
		// to avoid dupes....
		if len(base) == 16 && base[10:12] == "==" {
			base = base[:10]
		}

		var dt, err = time.Parse("2006-01-02", base)
		// Invalid issue directories are sometimes an error and sometimes something
		// to ignore due to how publishers sometimes name directories, how we flag
		// directories for review, etc.  We log a warning and move on, and
		// hopefully someday we have a more elegant approach.
		if err != nil {
			log.Printf("WARNING: Invalid issue directory %#v: %s", issuePath, err)
			continue
		}
		var issue = title.AppendIssue(dt, 1)
		cacheIssue(issue, issuePath)
	}

	return nil
}
