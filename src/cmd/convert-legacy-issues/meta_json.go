package main

import (
	"db"
	"encoding/json"
	"regexp"
	"schema"
	"strings"

	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
)

var validFilenameRegex = regexp.MustCompile(`^\d\d\d\d\.(pdf|tiff)$`)

// IssueMeta defines the structure we use in the JSON metadata
type IssueMeta struct {
	UUID          string   `json:"uuid"`
	GeneratedPath string   `json:"generated_path"`
	MARCOrgCode   string   `json:"marc_org_code,omitempty"`
	LCCN          string   `json:"lccn,omitempty"`
	Date          string   `json:"date,omitempty"`
	DateAsLabeled string   `json:"date_as_labeled,omitempty"`
	Topdir        string   `json:"topdir,omitempty"`
	SourceDatedir string   `json:"source_datedir,omitempty"`
	Volume        string   `json:"volume_number,omitempty"`
	Issue         string   `json:"issue_number,omitempty"`
	Edition       int      `json:"edition_number,omitempty"`
	EditionLabel  string   `json:"edition_label"`
	PageLabels    []string `json:"page_labels,omitempty"`
	SFTPDir       string   `json:"sftpdir,omitempty"`
}

func getDBIssue(path string) (*db.Issue, error) {
	var filename = filepath.Join(path, ".meta.json")

	if !fileutil.IsFile(filename) {
		return nil, fmt.Errorf("no .meta.json file found")
	}

	var meta, err = ParseMetaJSON(filename)
	if err != nil {
		return nil, err
	}

	var dbi *db.Issue
	dbi, err = db.FindIssueByLocation(path)
	if err != nil {
		return nil, err
	}
	if dbi == nil {
		dbi = &db.Issue{Location: path}
	}

	dbi.MARCOrgCode = meta.MARCOrgCode
	dbi.LCCN = meta.LCCN
	dbi.Date = meta.Date
	dbi.DateAsLabeled = meta.DateAsLabeled
	dbi.Volume = meta.Volume
	dbi.Issue = meta.Issue
	dbi.Edition = meta.Edition
	dbi.EditionLabel = meta.EditionLabel
	dbi.PageLabels = meta.PageLabels

	if meta.Date != "" {
		_, err = time.Parse("2006-01-02", meta.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid date value %q", meta.Date)
		}
	}

	// Check for invalid metadata
	if !db.ValidMOC(dbi.MARCOrgCode) {
		return nil, fmt.Errorf("invalid MARC Org Code %q", dbi.MARCOrgCode)
	}
	var title *db.Title
	title, err = db.FindTitleByLCCN(dbi.LCCN)
	if err != nil {
		return nil, err
	}
	if title == nil {
		return nil, fmt.Errorf("unable to find title for lccn %q", dbi.LCCN)
	}
	if dbi.Edition < 1 || dbi.Edition > 3 {
		return nil, fmt.Errorf("invalid edition number (%d)", dbi.Edition)
	}
	var si *schema.Issue
	si, err = dbi.SchemaIssue()
	if err != nil {
		return nil, err
	}

	// File iteration: count PDFs to see if page count is correct, look for any
	// tiffs so we can properly flag the issue as being from scanner, and check
	// filename patterns for validity
	var pdfCount int
	si.FindFiles()
	for _, f := range si.Files {
		// Let the .meta.json files live... for now.
		if f.Name == ".meta.json" {
			continue
		}

		if !validFilenameRegex.MatchString(f.Name) {
			return nil, fmt.Errorf("file %q doesn't match regex", f.Name)
		}

		var ext = strings.ToUpper(filepath.Ext(f.Name))
		switch ext {
		case ".TIFF":
			dbi.IsFromScanner = true
		case ".PDF":
			pdfCount++
		default:
			// This is redundant, but left here just in case we screw up the regex
			return nil, fmt.Errorf("unknown file extension %q", f.Name)
		}
	}

	if len(dbi.PageLabels) != pdfCount {
		return nil, fmt.Errorf("incorrect number of labels")
	}

	// Make sure the issues start out invisible to the front-end; until the jobs
	// run we don't want them being claimed by users
	dbi.WorkflowStep = schema.WSAwaitingProcessing

	return dbi, nil
}

// ParseMetaJSON reads the given JSON file to populate a new IssueMeta structure
func ParseMetaJSON(filename string) (*IssueMeta, error) {
	if !fileutil.IsFile(filename) {
		return nil, fmt.Errorf(`file "%s" does not exist`, filename)
	}

	var f, err = os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var dec = json.NewDecoder(f)
	var m IssueMeta
	err = dec.Decode(&m)
	if err != nil {
		return nil, err
	}

	return &m, nil
}
