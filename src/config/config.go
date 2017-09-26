// Package config is the project-specific configuration reader / parser /
// validator.  This uses the more generalized bashconf but adds our
// app-specific logic.
package config

import (
	"bashconf"
	"fileutil"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Config holds the configuration needed for this application to work.  Note
// that we don't (yet) pull in the entirety of the config data, just what's
// necessary for this project.
type Config struct {
	// DatabaseConnect is the all-in-one database connection value build from the
	// individual database settings
	DatabaseConnect string

	// OPJCompress stores the path to the openjpeg binary for creating JP2 files
	OPJCompress string `setting:"OPJ_COMPRESS"`

	// OPJDecompress stores the path to the openjpeg binary for reading JP2 files
	OPJDecompress string `setting:"OPJ_DECOMPRESS"`

	// GhostScript stores the path to the ghostscript binary for processing PDFs
	GhostScript string `setting:"GHOSTSCRIPT"`

	// Org code used for sftp-uploaded batches
	PDFBatchMARCOrgCode string `setting:"PDF_BATCH_MARC_ORG_CODE"`

	// Minimum number of pages an SFTPed issue must contain to be processed
	MinimumIssuePages int

	// DPI for generating JP2s
	DPI int

	// DPI used for images embedded in scanned PDFs, needed for ALTO XML
	ScannedPDFDPI int

	// JP2 quality value; converts to a rate using a an algorithm similar to that
	// which GraphicsMagick uses
	Quality float64

	// Paths to the various places we expect to find files
	MasterPDFUploadPath            string `setting:"MASTER_PDF_UPLOAD_PATH" type:"path"`
	MasterPDFBackupPath            string `setting:"MASTER_PDF_BACKUP_PATH" type:"path"`
	PDFIssuesAwaitingDerivatives   string `setting:"PDF_ISSUES_AWAITING_DERIVATIVES" type:"path"`
	PDFPageReviewPath              string `setting:"PDF_PAGE_REVIEW_PATH" type:"path"`
	PDFPagesAwaitingMetadataReview string `setting:"PDF_PAGES_AWAITING_METADATA_REVIEW" type:"path"`
	PDFPageSourcePath              string `setting:"PDF_PAGE_SOURCE_PATH" type:"path"`
	BatchOutputPath                string `setting:"BATCH_OUTPUT_PATH" type:"path"`
	PDFPageBackupPath              string `setting:"PDF_PAGE_BACKUP_PATH" type:"path"`
	ScansAwaitingDerivatives       string `setting:"SCANS_AWAITING_DERIVATIVES" type:"path"`

	// Eventually many of the paths above will be removed and this will be the
	// main location for all issues.  We'll have metadata in the database to tell
	// us workflow steps, rather relying on the filesystem paths.
	WorkflowPath string `setting:"WORKFLOW_PATH" type:"path"`
}

// Parse reads the given settings file and returns a parsed Config.  File paths
// are parsed and verified as they are used by most subsystems.  The database
// connection string is built, but is not tested.
func Parse(filename string) (*Config, error) {
	var c = &Config{}
	var errors []string

	var bc, err = bashconf.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Read in all the settings we've mapped with tags
	errors = append(errors, c.readTaggedFields(bc)...)

	// Database connection string: build it, but also make sure port is valid
	var i, _ = strconv.Atoi(bc["DB_PORT"])
	if i == 0 {
		errors = append(errors, "invalid DB_PORT")
	}
	c.DatabaseConnect = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", bc["DB_USER"],
		bc["DB_PASSWORD"], bc["DB_HOST"], bc["DB_PORT"], bc["DB_DATABASE"])

	c.MinimumIssuePages, _ = strconv.Atoi(bc["MINIMUM_ISSUE_PAGES"])
	if c.MinimumIssuePages == 0 {
		errors = append(errors, "invalid MINIMUM_ISSUE_PAGES: must be numeric and greater than 0")
	}

	c.DPI, _ = strconv.Atoi(bc["DPI"])
	if c.DPI < 72 {
		errors = append(errors, "invalid DPI: must be numeric and at least 72 (150 or higher is preferred)")
	}

	c.ScannedPDFDPI, _ = strconv.Atoi(bc["SCANNED_PDF_DPI"])
	if c.ScannedPDFDPI < 72 {
		errors = append(errors, "invalid DPI: must be numeric and at least 72")
	}

	c.Quality, _ = strconv.ParseFloat(bc["QUALITY"], 64)
	if c.Quality == 0 {
		errors = append(errors, "invalid QUALITY: must be numeric")
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("invalid configuration: %s", strings.Join(errors, ", "))
	}

	return c, nil
}

// readTaggedFields iterates over the tagged fields in c and pulls settings
// from bc.  If a tagged field has a type, it's used to process/validate the
// raw string value.
func (c *Config) readTaggedFields(bc bashconf.Config) (errors []string) {
	var rType = reflect.TypeOf(c).Elem()
	var rVal = reflect.ValueOf(c).Elem()

	for i := 0; i < rType.NumField(); i++ {
		var sf = rType.Field(i)

		// Ignore fields we can't set, regardless of tagging
		if !rVal.Field(i).CanSet() {
			continue
		}

		// If there's no "setting" tag, we have nothing to do here
		var sKey = sf.Tag.Get("setting")
		if sKey == "" {
			continue
		}

		var val = bc[sKey]
		var sType = sf.Tag.Get("type")
		switch sType {
		case "":
			rVal.Field(i).SetString(val)
		case "path":
			rVal.Field(i).SetString(val)
			if !fileutil.IsDir(val) {
				errors = append(errors, fmt.Sprintf("%#v (%#v) is not a directory", sKey, val))
				continue
			}
		}
	}

	return errors
}
