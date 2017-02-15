package sftp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// readdir wraps os.File's Readdir to handle common operations we need for
// getting a list of file info structures
func readdir(path string) ([]os.FileInfo, error) {
	var d *os.File
	var err error

	d, err = os.Open(path)
	if err != nil {
		return nil, err
	}

	var items []os.FileInfo
	items, err = d.Readdir(-1)
	d.Close()
	return items, err
}

// SFTPPDF stores a single PDF's data and error information
type SFTPPDF struct {
	Name    string
	RelPath string
	Error   error
}

// SFTPIssue stores a single issue's pdfs and any errors encountered
type SFTPIssue struct {
	Name    string
	RelPath string
	PDFs    []*SFTPPDF
	Error   error
}

// ScanPDFs reads in all issue PDFs and stores information about what's found,
// including definite errors and likely errors.  Returns an actual error object
// on fatal filesystem errors.
func (issue *SFTPIssue) ScanPDFs(path string) error {
	var items, err = readdir(path)
	if err != nil {
		return err
	}

	// Every item should be a PDF file
	for _, i := range items {
		var pdf = &SFTPPDF{Name: i.Name(), RelPath: filepath.Join(path, i.Name())}

		if !i.Mode().IsRegular() {
			pdf.Error = fmt.Errorf("regular file expected, got unexpected item instead")
		}

		var ext = strings.ToUpper(filepath.Ext(pdf.Name))
		if ext != ".PDF" {
			pdf.Error = fmt.Errorf("PDF file expected, got %s instead", ext)
		}

		issue.PDFs = append(issue.PDFs, pdf)
	}

	return nil
}

// SFTPPublisher stores uploaded file information and potential errors related
// to a publisher's SFTP directory
type SFTPPublisher struct {
	// Name is the directory where the publisher's uploaded files end up; e.g.,
	// /mnt/news/sftp/dailynews would have a name of "dailynews"
	Name string

	// Path is the location of the publisher directory as found by scanning for
	// publishers under the configured SFTP location.  RealPath is the target if
	// Path is a symlink, otherwise it's equal to Path.
	Path, RealPath string

	// Issues holds information about the per-issue subdirectories
	Issues []*SFTPIssue
}

// ScanIssues reads in all issue directories and files for a publisher, and
// stores information about what's found, including definite errors and likely
// errors.  Returns an actual error object on fatal filesystem errors.  This
// should only be run on a publisher with its path data already set up.
func (p *SFTPPublisher) ScanIssues() error {
	var items, err = readdir(p.RealPath)
	if err != nil {
		return err
	}

	// Every item should be a properly formatted date directory which we can turn
	// into an SFTPIssue
	for _, i := range items {
		var issue = &SFTPIssue{}
		issue.Name = i.Name()
		issue.RelPath = filepath.Join(p.Name, i.Name())

		if !i.IsDir() {
			issue.Error = fmt.Errorf("folder expected, got file instead")
		}

		p.Issues = append(p.Issues, issue)
	}

	return nil
}

// BuildPublishers takes an SFTP root directory and returns all the directories
// one level below.  Since it's impossible for a publisher to do anything at
// this level, we don't try to find or report any non-filesystem errors.
func BuildPublishers(path string) ([]*SFTPPublisher, error) {
	var pubList []*SFTPPublisher
	var items, err = readdir(path)
	if err != nil {
		return nil, err
	}

	for _, i := range items {
		var pubName = i.Name()
		var path = filepath.Join(path, pubName)
		var realPath = path
		if i.Mode()&os.ModeSymlink != 0 {
			realPath, err = os.Readlink(path)
			if err != nil {
				return nil, err
			}
			i, err = os.Stat(realPath)
			if err != nil {
				return nil, err
			}
		}
		realPath = filepath.Clean(realPath)

		// Skip anything we can't descend into
		if !i.IsDir() && i.Mode()&os.ModeSymlink == 0 {
			continue
		}
		var p = &SFTPPublisher{Name: pubName}
		p.Path = path
		p.RealPath = realPath
		pubList = append(pubList, p)
	}

	return pubList, nil
}
