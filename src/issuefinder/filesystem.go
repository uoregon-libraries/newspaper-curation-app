package issuefinder

import (
	"chronam"
	"fileutil"
	"fmt"
	"path/filepath"
	"schema"
	"strconv"
	"strings"
	"time"
)

// FindSFTPIssues is just barely its own special case because unlike the
// standard structure, there is no "topdir" element in the paths
func (f *Finder) FindSFTPIssues(path string) error {
	// First find all titles
	var titlePaths, err = fileutil.FindDirectories(path)
	if err != nil {
		return err
	}

	// Find all issues next
	for _, titlePath := range titlePaths {
		err = f.findStandardIssuesForTitlePath(titlePath, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// FindStandardIssues does the work of finding and returning all issue
// information within a given path with the assumption that the path conforms
// to `topdir/sftpnameOrLCCN/yyyy-mm-dd/`
func (f *Finder) FindStandardIssues(path string) error {
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
		err = f.findStandardIssuesForTitlePath(titlePath, false)
		if err != nil {
			return err
		}
	}

	return nil
}

// findStandardIssuesForTitle finds all issues within the given title's path by
// looking for YYYY-MM-DD or YYYY-MM-DD_EE formatted directories.  The latter
// format is only allowed if strict is false (SFTP issues, for instance, don't
// allow an edition).  As the path is expected to be "standard", the last
// directory element in the path must be an SFTP title name or an LCCN.
func (f *Finder) findStandardIssuesForTitlePath(titlePath string, strict bool) error {
	// Make sure we have a legitimate title - we have to check titles by
	// directory and LCCN
	var titleName = filepath.Base(titlePath)
	var title = f.findFilesystemTitle(titleName, titlePath)

	// A missing title is a problem for all standard directory layouts, because
	// these are always in-house issues.  Live batches or old batches on the
	// filesystem wouldn't hit this check.
	//
	// Note that despite not having a valid title we still scan the directory in
	// order to catch other errors and aggregate the unknown titles' issues.
	if title == nil {
		title = &schema.Title{LCCN: titlePath}
		f.newError(titlePath, fmt.Errorf("unable to find title %#v in database", titleName))
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
		var addErr = func(e error) { errors = append(errors, f.newError(issuePath, e).SetTitle(title)) }

		// A suffix of "-error" is a manually flagged error; we should keep an eye
		// on these, but their contents can still be valuable
		if strings.HasSuffix(base, "-error") {
			addErr(fmt.Errorf("manually flagged issue"))
			base = base[:len(base)-6]
		}

		// Check for an edition suffix
		var edition = 1
		if len(base) >= 13 && base[10] == '_' {
			var edstr = base[11:13]
			edition, err = strconv.Atoi(edstr)
			if edition < 1 {
				addErr(fmt.Errorf("invalid issue directory edition suffix (%s)", edstr))
			}

			// SFTP dirs can't have an edition suffix, so we conditionally store an error
			if strict {
				addErr(fmt.Errorf("edition suffix isn't allowed here"))
			}
			base = base[:10] + base[13:]
		}

		// And of course we have to remove our wonderful path hack that was built
		// to avoid dupes....
		if len(base) == 16 && base[10:12] == "==" {
			base = base[:10]
		}

		var dt, err = time.Parse("2006-01-02", base)
		// Invalid issue directory names can't have an issue, so we can continue
		// without fixing up the errors
		if err != nil {
			addErr(fmt.Errorf("invalid issue directory name: must be formatted YYYY-MM-DD"))
			continue
		}

		var issue = title.AddIssue(&schema.Issue{Date: dt, Edition: edition, Location: issuePath})

		issue.FindFiles()

		for _, e := range errors {
			e.SetIssue(issue)
		}
		f.Issues = append(f.Issues, issue)
		f.verifyStandardIssueFiles(issue, strict)
	}

	return nil
}

// verifyStandardIssueFiles looks for errors in any files within a given issue.
// In our standard layout, the following are considered errors:
// - There are files that aren't regular (symlinks, directories, etc), though
//   some exceptions exist, such as the .derivatives sub-directory
// - There are files that aren't pdf, tiff, jp2, or xml (though a
//   few exceptions exist, such as .meta.json and Adobe Bridge dot-files we
//   ignore when we get to the processing phase)
// - Any derivative file exists without a corresponding PDF
// - The issue directory is empty
//
// Additionally, if strict is true, we don't allow for any exceptions to the
// file type and extension rules, to prevent SFTP directories from being
// processed when there's anything non-conformant.
func (f *Finder) verifyStandardIssueFiles(issue *schema.Issue, strict bool) {
	if len(issue.Files) == 0 {
		f.newError(issue.Location, fmt.Errorf("no issue files found")).SetIssue(issue)
		return
	}

	// Cache all filenames beforehand
	var hasPDF = make(map[string]bool)
	for _, file := range issue.Files {
		var ext = strings.ToLower(filepath.Ext(file.Name))
		if ext == ".pdf" {
			hasPDF[strings.Replace(file.Name, ext, "", 1)] = true
		}
	}

	// NOTE: These rules *seem* general-case, but they only makes sense for
	// standard issues, so if we extract this code into a more reusable function,
	// we need to refine or separate some rules.  A batch can have the master
	// PDFs stored in a subdirectory, a meta.json file (not .meta.json), and
	// issue-level XMLs that don't have a corresponding PDF.
	for _, file := range issue.Files {
		// We could check .meta.json, .derivatives, .Bridge*, etc. individually,
		// but the very low likelihood of dot-files being real errors just isn't
		// worth the granularity.
		if strict == false && file.Name[0] == '.' {
			continue
		}

		var makeErr = func(format string, args ...interface{}) {
			f.newError(file.Location, fmt.Errorf(format, args...)).SetFile(file)
		}
		if file.IsDir() {
			makeErr("%q is a subdirectory", file.Name)
			continue
		}

		if !file.IsRegular() {
			makeErr("%q is not a regular file", file.Name)
			continue
		}

		// NOTE: It may be a good idea to validate that all content files are
		// numeric-only.  Mostly ####.jp2/pdf/etc, though if we handle batches at
		// some point, we may also see YYYYMMDDEE.xml.
		var ext = strings.ToLower(filepath.Ext(file.Name))
		if ext != ".pdf" && ext != ".tiff" && ext != ".tif" && ext != ".jp2" && ext != ".xml" {
			makeErr("%q has an invalid extension", file.Name)
			continue
		}

		if ext != ".pdf" {
			if !hasPDF[strings.Replace(file.Name, ext, "", 1)] {
				makeErr("%q has no associated PDF", file.Name)
				continue
			}
		}
	}
}

// FindDiskBatches finds all batches in the batch output path, then finds their
// titles and their titles' issues, and caches everything
func (f *Finder) FindDiskBatches(path string) error {
	// First, find batch directories
	var batchDirs, err = fileutil.FindDirectories(path)
	if err != nil {
		return err
	}

	// For each batch, we want to store the batch information as well as
	// everything in it
	for _, batchDir := range batchDirs {
		// To simplify things, we don't actually scour the filesystem for titles
		// and issues; instead, we parse the batch XML, as that should *always*
		// contain all issues (and their titles LCCNs).
		f.cacheBatchDataFromXML(batchDir)
	}

	return nil
}

// cacheBatchDataFromXML reads the batch.xml file and caches all titles and
// issues found inside.  Errors are stored, and many are ignored, as a broken
// batch or batch XML isn't necessarily uncommon with live data, oddly enough.
// We don't bother to verify issue directories or files at this point, because
// only a code bug would cause the generated batches to break, which isn't
// something anybody but a dev can deal with.
func (f *Finder) cacheBatchDataFromXML(batchDir string) {
	var parts = strings.Split(batchDir, string(filepath.Separator))
	var batchName = parts[len(parts)-1]
	var batch, err = schema.ParseBatchname(batchName)
	if err != nil {
		f.newError(batchDir, fmt.Errorf("invalid batch directory name %#v: %s", batchDir, err))
		return
	}
	batch.Location = batchDir
	f.Batches = append(f.Batches, batch)

	var bx *chronam.BatchXML
	bx, err = chronam.ParseBatchXML(batchDir)
	if err != nil {
		f.newError(batchDir, fmt.Errorf("unable to process batch XML: %s", err)).SetBatch(batch)
		return
	}

	var dataDir = filepath.Join(batchDir, "data")

	// All titles within a batch are treated as being unique within the system as
	// a whole.  And since batched titles may or may not be in our database, and
	// we always know we have an LCCN, we don't bother doing global lookups.
	for _, ix := range bx.Issues {
		var dt time.Time
		dt, err = time.Parse("2006-01-02", ix.Date)
		if err != nil {
			f.newError(batchDir, fmt.Errorf("invalid issue date in batch XML (%#v): %s", ix, err)).SetBatch(batch)
			return
		}
		var ed int
		ed, err = strconv.Atoi(ix.EditionOrder)
		if err != nil {
			f.newError(batchDir, fmt.Errorf("invalid issue edition in batch XML (%#v)", ix)).SetBatch(batch)
		}

		var titleDir = filepath.Join(dataDir, ix.LCCN)
		var title = f.findOrCreateUnknownFilesystemTitle(ix.LCCN, titleDir)
		var issueDir = filepath.Join(dataDir, ix.Content)
		var issue = title.AddIssue(&schema.Issue{Date: dt, Edition: ed, Location: issueDir})
		batch.AddIssue(issue)
		title.AddIssue(issue)
		f.Issues = append(f.Issues, issue)
	}
}
