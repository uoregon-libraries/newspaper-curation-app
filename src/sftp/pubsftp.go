// Package sftp exposes types and minimal logic for finding SFTP publishers'
// directories and their uploaded issues
package sftp

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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

// byName implements sort.Interface for sorting os.FileInfo data by name
type byName []os.FileInfo

func (n byName) Len() int           { return len(n) }
func (n byName) Swap(i, j int)      { n[i], n[j] = n[j], n[i] }
func (n byName) Less(i, j int) bool { return n[i].Name() < n[j].Name() }

// PDF stores a single PDF's data and error information.  Note that this can be
// used for non-PDF files since we just enumerate over everything in an issue
// directory.  In these cases, Error will indicate that the "PDF" had a non-PDF
// extension or was a non-regular file (sym link, directory, etc.)
type PDF struct {
	Name         string
	RelativePath string
	Error        error
	Modified     time.Time
	Issue        *Issue
}

// Issue stores a single issue's pdfs and any errors encountered.  An "issue"
// is really anything found at the publisher's root directory, and therefore
// could be a non-issue directory, a file, etc.  In these cases, Error will
// explain that the "Issue" isn't what we expected an issue to be.
type Issue struct {
	Name         string
	RelativePath string
	PDFs         []*PDF
	Error        error
	Modified     time.Time
	Publisher    *Publisher
}

// ScanPDFs reads in all issue PDFs and stores information about what's found,
// including definite errors and likely errors.  Returns an actual error object
// on fatal filesystem errors.
func (issue *Issue) ScanPDFs() error {
	var path = filepath.Join(issue.Publisher.RealPath, issue.Name)
	var items, err = readdir(path)
	if err != nil {
		return err
	}

	// Every item should be a PDF file
	sort.Sort(byName(items))
	for _, i := range items {
		var pdf = &PDF{
			Name:         i.Name(),
			RelativePath: filepath.Join(issue.RelativePath, i.Name()),
			Modified:     i.ModTime(),
			Issue:        issue,
		}

		if !i.Mode().IsRegular() {
			pdf.Error = fmt.Errorf("regular file expected, got unexpected item instead")
		}

		var ext = strings.ToUpper(filepath.Ext(pdf.Name))
		if ext != ".PDF" {
			pdf.Error = fmt.Errorf("PDF file expected, got %s instead", ext)
		}

		if issue.Modified.Before(pdf.Modified) {
			issue.Modified = pdf.Modified
		}
		issue.PDFs = append(issue.PDFs, pdf)
	}

	return nil
}

// Publisher stores uploaded file information and potential errors related
// to a publisher's SFTP directory
type Publisher struct {
	// Name is the directory where the publisher's uploaded files end up; e.g.,
	// /mnt/news/sftp/dailynews would have a name of "dailynews"
	Name string

	// Path is the location of the publisher directory as found by scanning for
	// publishers under the configured SFTP location.  RealPath is the target if
	// Path is a symlink, otherwise it's equal to Path.
	Path, RealPath string

	// Issues holds information about the per-issue subdirectories
	Issues []*Issue
}

// ScanIssues reads in all issue directories and files for a publisher, and
// stores information about what's found, including definite errors and likely
// errors.  Returns an actual error object on fatal filesystem errors.  This
// should only be run on a publisher with its path data already set up.
func (p *Publisher) ScanIssues() error {
	var items, err = readdir(p.RealPath)
	if err != nil {
		return err
	}

	// Every item should be a properly formatted date directory which we can turn
	// into an Issue
	sort.Sort(byName(items))
	for _, i := range items {
		var issue = &Issue{Publisher: p}
		issue.Name = i.Name()
		issue.RelativePath = filepath.Join(p.Name, i.Name())

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
func BuildPublishers(path string) ([]*Publisher, error) {
	var pubList []*Publisher
	var items, err = readdir(path)
	if err != nil {
		return nil, err
	}

	sort.Sort(byName(items))
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
		var p = &Publisher{Name: pubName}
		p.Path = path
		p.RealPath = realPath
		pubList = append(pubList, p)
	}

	return pubList, nil
}
