// meta_json.go handles parsing .meta.json to get more valid issue metadata

package schema

import (
	"encoding/json"
	"fileutil"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// IssueMeta defines the structure we use in the JSON metadata.  We don't use
// most of this in this app, but it's all here for the sake of completeness
type IssueMeta struct {
	UUID          string    `json:"uuid"`
	GeneratedPath string    `json:"generated_path"`
	MARCOrgCode   string    `json:"marc_org_code,omitempty"`
	LCCN          string    `json:"lccn,omitempty"`
	Date          string    `json:"date,omitempty"`
	DateAsLabeled string    `json:"date_as_labeled,omitempty"`
	Topdir        string    `json:"topdir,omitempty"`
	SourceDatedir string    `json:"source_datedir,omitempty"`
	Volume        string    `json:"volume_number,omitempty"`
	Issue         string    `json:"issue_number,omitempty"`
	Edition       int       `json:"edition_number,omitempty"`
	EditionLabel  string    `json:"edition_label"`
	PageLabels    []string  `json:"page_labels,omitempty"`
	SFTPDir       string    `json:"sftpdir,omitempty"`
	RawDate       time.Time `json:"-"`
}

// ParseMetadata reads the issue's .meta.json or meta.json to give the issue a
// valid date and do some basic validity checking.
//
// TODO: Also read XML metadata if present, as it's possible the two could get
// out of sync, in which case we'd want to report an error.
func (i *Issue) ParseMetadata() error {
	var filename = filepath.Join(i.Location, ".meta.json")

	if !fileutil.IsFile(filename) {
		filename = filepath.Join(i.Location, "meta.json")
		// If *that* file doesn't exist, we don't parse anything; this is a legacy
		// issue or an unprocessed issue, and we just assume the parsed directory
		// structure is correct for now
		if !fileutil.IsFile(filename) {
			return nil
		}
	}

	var meta, err = ParseMetaJSON(filename)
	if err != nil {
		return err
	}

	if meta.LCCN != i.Title.LCCN {
		return fmt.Errorf("LCCN doesn't match Title's LCCN")
	}

	if meta.UUID == "" {
		return fmt.Errorf("missing uuid key")
	}

	var dt time.Time
	dt, err = time.Parse("2006-01-02", meta.Date)
	if err != nil {
		return fmt.Errorf("invalid date value <%s>", meta.Date)
	}

	i.Date = dt

	return nil
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
