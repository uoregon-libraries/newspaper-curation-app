package main

import (
	"encoding/xml"
	"fileutil"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"schema"
	"strconv"
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
	err = cacheBatches()
	if err != nil {
		log.Fatalf("Error trying to cache batches: %s", err)
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
		cacheFilesystemIssue(issue, issuePath, nil)
	}

	return nil
}

// cacheBatches finds all batches in the batch output path, then finds their
// titles and their titles' issues, and caches everything
func cacheBatches() error {
	// First, find batch directories
	var batchDirs, err = fileutil.FindDirectories(Conf.BatchOutputPath)
	if err != nil {
		return err
	}

	// For each batch, we want to store the batch information as well as
	// everything in it
	for _, batchDir := range batchDirs {
		// To simplify things, we don't actually scour the filesystem for titles
		// and issues; instead, we parse the batch XML, as that should *always*
		// contain all issues (and their titles LCCNs).
		err = cacheBatchDataFromXML(batchDir)
		if err != nil {
			return err
		}
	}

	return nil
}

// batchXML is used to deserialize batch.xml files to get at their issues list
type batchXML struct {
	XMLName xml.Name   `xml:"batch"`
	Issues  []issueXML `xml:"issue"`
}

// issueXML describes each <issue> element in the batch XML
type issueXML struct {
	EditionOrder string `xml:"editionOrder,attr"`
	Date         string `xml:"issueDate,attr"`
	LCCN         string `xml:"lccn,attr"`
	Content      string `xml:",innerxml"`
}

// cacheBatchDataFromXML reads the batch.xml file and caches all titles and
// issues found inside
func cacheBatchDataFromXML(batchDir string) error {
	var parts = strings.Split(batchDir, string(filepath.Separator))
	var batchName = parts[len(parts)-1]
	var batch, err = schema.ParseBatchname(batchName)
	if err != nil {
		return fmt.Errorf("batch directory %#v isn't valid: %s", batchDir, err)
	}

	var xmlFile = filepath.Join(batchDir, "data", "batch.xml")
	if !fileutil.IsFile(xmlFile) {
		return fmt.Errorf("batch directory %#v has no batch.xml", batchDir)
	}

	var contents []byte
	contents, err = ioutil.ReadFile(xmlFile)
	if err != nil {
		return fmt.Errorf("batch XML file (%#v) can't be read: %s", xmlFile, err)
	}

	var bx batchXML
	err = xml.Unmarshal(contents, &bx)
	if err != nil {
		return fmt.Errorf("unable to unmarshal batch XML %#v: %s", xmlFile, err)
	}

	for _, ix := range bx.Issues {
		var dt time.Time
		dt, err = time.Parse("2006-01-02", ix.Date)
		if err != nil {
			return fmt.Errorf("invalid issue date in batch XML %#v: %s (issue dump: %#v)", xmlFile, err, ix)
		}
		var ed int
		ed, err = strconv.Atoi(ix.EditionOrder)
		if err != nil {
			return fmt.Errorf("invalid edition number in batch XML %#v: %s (issue dump: %#v)", xmlFile, err, ix)
		}
		var title = findOrCreateTitle(ix.LCCN)
		var issue = title.AppendIssue(dt, ed)
		cacheFilesystemIssue(issue, filepath.Join(batchDir, ix.Content), batch)
	}

	return nil
}
