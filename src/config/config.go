// Package config is the project-specific configuration reader / parser /
// validator.  This uses the more generalized bashconf but adds our
// app-specific logic.
package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/uoregon-libraries/gopkg/bashconf"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/datasize"
)

// Config holds the configuration needed for this application to work
type Config struct {
	// DatabaseConnect is the all-in-one database connection value built from the
	// individual database settings
	DatabaseConnect string

	// We pull the DB string values manually, but having this already converted
	// to int is easier
	DBPort int `setting:"DB_PORT" type:"int"`

	// SFTPGo settings
	SFTPGoEnabled      bool
	SFTPGoAPIURL       *url.URL
	SFTPGoAdminLogin   string `setting:"SFTPGO_ADMIN_LOGIN"`
	SFTPGoAdminAPIKey  string `setting:"SFTPGO_ADMIN_API_KEY"`
	SFTPGoNewUserQuota datasize.Datasize

	// Binary paths
	GhostScript   string `setting:"GHOSTSCRIPT"`
	OPJCompress   string `setting:"OPJ_COMPRESS"`
	OPJDecompress string `setting:"OPJ_DECOMPRESS"`

	// Web configuration
	Webroot            string `setting:"WEBROOT" type:"url"`
	BindAddress        string `setting:"BIND_ADDRESS"`
	IIIFBaseURL        string `setting:"IIIF_BASE_URL" type:"url"`
	NewsWebroot        string `setting:"NEWS_WEBROOT" type:"url"`
	StagingNewsWebroot string `setting:"STAGING_NEWS_WEBROOT" type:"url"`

	// MARC location(s) for getting XML for unknown titles
	MARCLocation1 string `setting:"MARC_LOCATION_1"`
	MARCLocation2 string `setting:"MARC_LOCATION_2"`

	// Paths to the various places we expect to find files
	PDFUploadPath        string `setting:"PDF_UPLOAD_PATH" type:"path"`
	ScanUploadPath       string `setting:"SCAN_UPLOAD_PATH" type:"path"`
	PDFBackupPath        string `setting:"ORIGINAL_PDF_BACKUP_PATH" type:"path"`
	PDFPageReviewPath    string `setting:"PDF_PAGE_REVIEW_PATH" type:"path"`
	BatchOutputPath      string `setting:"BATCH_OUTPUT_PATH" type:"path"`
	WorkflowPath         string `setting:"WORKFLOW_PATH" type:"path"`
	ErroredIssuesPath    string `setting:"ERRORED_ISSUES_PATH" type:"path"`
	IssueCachePath       string `setting:"ISSUE_CACHE_PATH" type:"path"`
	AppRoot              string `setting:"APP_ROOT" type:"path"`
	METSXMLTemplatePath  string `setting:"METS_XML_TEMPLATE_PATH" type:"file"`
	BatchXMLTemplatePath string `setting:"BATCH_XML_TEMPLATE_PATH" type:"file"`

	// Issue processor / batch maker rules
	MinimumIssuePages   int    `setting:"MINIMUM_ISSUE_PAGES" type:"int"`
	PDFBatchMARCOrgCode string `setting:"PDF_BATCH_MARC_ORG_CODE"`
	MaxBatchSize        int    `setting:"MAX_BATCH_SIZE" type:"int"`
	MinBatchSize        int    `setting:"MIN_BATCH_SIZE" type:"int"`

	// Derivative generation rules
	DPI           int     `setting:"DPI" type:"int"`
	Quality       float64 `setting:"QUALITY" type:"float"`
	ScannedPDFDPI int     `setting:"SCANNED_PDF_DPI" type:"int"`
}

// Parse reads the given settings file and returns a parsed Config.  File paths
// are parsed and verified as they are used by most subsystems.  The database
// connection string is built, but is not tested.
func Parse(filename string) (*Config, error) {
	var c = &Config{}
	var errors []string

	var bc = bashconf.New()
	bc.EnvironmentPrefix("NCA_")
	var err = bc.ParseFile(filename)
	if err != nil {
		return nil, err
	}
	err = bc.Store(c)
	if err != nil {
		return nil, err
	}

	// Database connection string: build it, but also make sure port is valid
	c.DatabaseConnect = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", bc.Get("DB_USER"),
		bc.Get("DB_PASSWORD"), bc.Get("DB_HOST"), c.DBPort, bc.Get("DB_DATABASE"))

	// SFTPGo API URL is special: we want an actual URL out of it, but we also
	// want to allow it to be empty via "" or "-"
	c.SFTPGoAPIURL, err = parseOptionalURL(bc.Get("SFTPGO_API_URL"))
	if err != nil {
		errors = append(errors, fmt.Sprintf("invalid SFTPGoAPIURL: %s", err))
	}
	c.SFTPGoEnabled = (c.SFTPGoAPIURL != nil)

	// We validate the quota here rather than just reading it as a string - then
	// end-users could get hit with errors just because they used the default
	var quota = bc.Get("SFTPGO_NEW_USER_QUOTA")
	c.SFTPGoNewUserQuota, err = datasize.New(quota)
	if err != nil {
		errors = append(errors, fmt.Sprintf("invalid SFTPGoNewUserQuota: %s", err))
	}

	if c.MinimumIssuePages < 1 {
		errors = append(errors, "invalid MINIMUM_ISSUE_PAGES: must be numeric and greater than 0")
	}

	if c.DPI < 72 {
		errors = append(errors, "invalid DPI: must be numeric and at least 72 (150 or higher is preferred)")
	}

	if c.ScannedPDFDPI < 72 {
		errors = append(errors, "invalid DPI: must be numeric and at least 72")
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("invalid configuration: %s", strings.Join(errors, ", "))
	}

	return c, nil
}

func parseOptionalURL(val string) (*url.URL, error) {
	if val == "" || val == "-" {
		return nil, nil
	}

	var u, err = url.Parse(val)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("scheme must be http or https")
	}

	if u.Host == "" {
		return nil, fmt.Errorf("host may not be empty")
	}

	return u, nil
}
