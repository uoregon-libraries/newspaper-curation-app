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
		err = f.findStandardIssuesForTitlePath(titlePath, false)
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
		err = f.findStandardIssuesForTitlePath(titlePath, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// findStandardIssuesForTitle finds all issues within the given title's path by
// looking for YYYY-MM-DD or YYYY-MM-DD_EE formatted directories.  The latter
// format is only alled if allowEdition is true (which is not the case for SFTP
// issues, for instance).  As the path is expected to be "standard", the last
// directory element in the path must be an SFTP title name or an LCCN.
func (f *Finder) findStandardIssuesForTitlePath(titlePath string, allowEdition bool) error {
	// Make sure we have a legitimate title - we have to check titles by
	// directory and LCCN
	var titleName = filepath.Base(titlePath)
	var title = f.findTitle(titleName)

	// A missing title is a problem for all standard directory layouts, because
	// these are always in-house issues.  Live batches or old batches on the
	// filesystem wouldn't hit this check.
	//
	// Note that despite a valid title we still scan the directory in order to
	// catch other errors and aggregate the unknown titles' issues.
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
			if !allowEdition {
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

		var issue = &schema.Issue{Title: title, Date: dt, Edition: edition, Location: issuePath}
		for _, e := range errors {
			e.SetIssue(issue)
		}
		f.Issues = append(f.Issues, issue)
	}

	return nil
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

		var title = f.findTitle(ix.LCCN)
		var issue = &schema.Issue{Title: title, Date: dt, Edition: ed, Location: filepath.Join(batchDir, ix.Content)}
		batch.AddIssue(issue)
		f.Issues = append(f.Issues, issue)
	}
}
