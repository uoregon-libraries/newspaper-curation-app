package uploadedissuehandler

import (
	"db"
	"fmt"
	"html/template"
	"issuefinder"
	"issuesearch"
	"jobs"

	"path/filepath"
	"schema"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/pdf"
)

// Errors wraps an array of error strings for nicer display
type Errors []template.HTML

func (e Errors) String() string {
	var sList = make([]string, len(e))
	for i, s := range e {
		sList[i] = string(s)
	}
	return strings.Join(sList, "; ")
}

// TitleType tells us if a title contains born-digital issues or scanned
type TitleType int

// The two constants for TitleType
const (
	TitleTypeScanned TitleType = iota
	TitleTypeBornDigital
)

// String tells us a human-friendly meaning for the type of title
func (tt TitleType) String() string {
	switch tt {
	case TitleTypeScanned:
		return "Scanned"
	case TitleTypeBornDigital:
		return "BornDigital"
	}

	return "N/A"
}

// Title wraps a schema.Title with some extra information for web presentation.
type Title struct {
	*schema.Title
	Slug        string
	allErrors   *issuefinder.ErrorList
	Errors      Errors
	TitleErrors int
	ChildErrors int
	TotalErrors int
	Issues      []*Issue
	IssueLookup map[string]*Issue
	Type        TitleType
}

func (t *Title) decorateIssues(issueList []*schema.Issue) {
	t.Issues = make([]*Issue, 0)
	t.IssueLookup = make(map[string]*Issue)
	for _, i := range issueList {
		if !searcher.IsInProcess(i.Key()) {
			t.appendSchemaIssue(i)
		}
	}
}

func (t *Title) appendSchemaIssue(i *schema.Issue) *Issue {
	var issue = &Issue{Issue: i, Slug: i.DateEdition(), Title: t}
	issue.decorateFiles(i.Files)
	issue.decorateErrors()
	issue.decorateExternalErrors()
	t.Issues = append(t.Issues, issue)
	t.IssueLookup[issue.Slug] = issue

	return issue
}

func (t *Title) addError(err template.HTML) {
	t.Errors = append(t.Errors, err)
	t.TitleErrors++
	t.TotalErrors++
}

func (t *Title) decorateErrors() {
	t.Errors = make(Errors, 0)
	for _, e := range t.allErrors.TitleErrors[t.Title] {
		t.addError(safeError(e.Error.Error()))
	}
}

// AddChildError should be called when an issue has any kind of error so we
// know this title's issues will need to be looked at closely
func (t *Title) AddChildError() {
	t.ChildErrors++
	t.TotalErrors++
}

// We want HTML-friendly errors for when we need to put in our own, but
// we don't necessarily trust that the more internal errors won't have
// things like "<" in 'em, so... this happens.
func safeError(err string) template.HTML {
	return template.HTML(template.HTMLEscapeString(err))
}

// Show returns true if the title has any issues or errors.  If there are no
// errors and no issues, there's no reason to display it.
func (t *Title) Show() bool {
	return len(t.Issues) > 0 || len(t.Errors) > 0
}

// Link returns a link for this title
func (t *Title) Link() template.HTML {
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, TitlePath(t.Slug), t.Name))
}

// Issue wraps a schema.Issue for web presentation
type Issue struct {
	*schema.Issue
	Slug        string           // Short, URL-friendly identifier for an issue
	Title       *Title           // Title to which this issue belongs
	QueueInfo   template.HTML    // Informational message from the queue process, if any
	Errors      Errors           // List of errors automatically identified for this issue
	ChildErrors int              // Count of child errors for use in the templates
	TotalErrors int              // Count of child + issue errors
	Files       []*File          // List of files
	FileLookup  map[string]*File // Lookup for finding a File by its filename / slug
	Modified    time.Time        // When this issue's most recent file was modified
}

func (i *Issue) decorateFiles(fileList []*schema.File) {
	i.Files = make([]*File, 0)
	i.FileLookup = make(map[string]*File)
	for _, f := range fileList {
		i.appendSchemaFile(f)
		if i.Modified.Before(f.ModTime) {
			i.Modified = f.ModTime
		}
	}
}

func (i *Issue) appendSchemaFile(f *schema.File) {
	var slug = filepath.Base(f.Location)
	var pdf = &File{File: f, Slug: slug, Issue: i}
	pdf.decorateErrors()
	i.Files = append(i.Files, pdf)
	i.FileLookup[pdf.Slug] = pdf
}

func (i *Issue) addError(err template.HTML) {
	i.Errors = append(i.Errors, err)
	i.Title.AddChildError()
	i.TotalErrors++
}

func (i *Issue) decorateErrors() {
	i.Errors = make(Errors, 0)
	for _, e := range i.Title.allErrors.IssueErrors[i.Issue] {
		i.addError(safeError(e.Error.Error()))
	}
}

// AddChildError should be called when a file has any kind of error so we know
// this issue's pages will need to be looked at closely
func (i *Issue) AddChildError() {
	i.ChildErrors++
	if i.ChildErrors == 1 {
		i.addError(safeError("one or more files are invalid"))
		return
	}
	i.TotalErrors++
}

// decorateExternalErrors checks for external problems we don't detect when
// just scanning the issue directories and files
func (i *Issue) decorateExternalErrors() {
	i.decorateDupeErrors()
	i.decoratePriorJobLogs()
}

// decorateDupeErrors adds errors to the issue if we find the same key in the global watcher
func (i *Issue) decorateDupeErrors() {
	var key, err = issuesearch.ParseSearchKey(i.Key())
	// This shouldn't be able to happen, but if it does the best we can do is log
	// it and skip dupe checking; better than panicking in the lookup below
	if err != nil {
		logger.Errorf("Invalid issue key %q", i.Key())
		return
	}

	var watcherIssues = watcher.Scanner.LookupIssues(key)
	for _, wi := range watcherIssues {
		if wi.WorkflowStep == i.WorkflowStep {
			continue
		}

		var errstr = fmt.Sprintf("likely duplicate of %s", wi.WorkflowIdentification())
		if wi.WorkflowStep == schema.WSInProduction {
			errstr = fmt.Sprintf(`likely duplicate of a live issue: <a href="%s">%s, %s</a>`,
				wi.Location[:len(wi.Location)-5], wi.Title.Name, wi.RawDate)
		}
		i.addError(template.HTML(errstr))
	}
}

// decoratePriorJobLogs adds information to issues that have old failed jobs.
func (i *Issue) decoratePriorJobLogs() {
	var dbi, err = db.FindIssueByKey(i.Key())
	if err != nil {
		logger.Errorf("Unable to look up issue for decorating queue messages: %s", err)
		return
	}
	if dbi == nil {
		return
	}

	var dbJobs []*db.Job
	dbJobs, err = db.FindJobsForIssueID(dbi.ID)
	if err != nil {
		logger.Errorf("Unable to look up jobs for issue id %d (%q): %s", dbi.ID, i.Key(), err)
		return
	}

	var subErrors []string
	for _, j := range dbJobs {
		// We only care to report on the failed jobs, as those haven't been requeued
		if j.Status != string(jobs.JobStatusFailed) {
			continue
		}

		for _, log := range j.Logs() {
			switch log.LogLevel {
			case "DEBUG", "INFO", "WARN":
				continue
			case "ERROR", "CRIT":
				subErrors = append(subErrors, log.Message)
			default:
				logger.Errorf("Unknown job log level: %q", log.LogLevel)
			}
		}
	}

	if len(subErrors) > 0 {
		var listItems string
		for _, e := range subErrors {
			listItems += "<li>" + e + "</li>\n"
		}
		var msg = fmt.Sprintf(`
			A previous queue attempt failed, but you can attempt to re-queue or
			contact the system administrator.
			<br /><br />
			Details:
			<ul>
				%s
			</ul>
			`, listItems)
		i.QueueInfo = template.HTML(msg)
	}
}

// IsNew tells the presentation if the issue is fairly new, which can be
// important for some publishers who upload over several days
func (i *Issue) IsNew() bool {
	return time.Since(i.Modified) < time.Hour*24*14
}

// IsDangerouslyNew on the other hand tells us if the issue is so new that
// we're not okay with manual queueing even with a warning, because it's just
// not safe!
func (i *Issue) IsDangerouslyNew() bool {
	return time.Since(i.Modified) < time.Hour*24*2
}

// Link returns a link for this title
func (i *Issue) Link() template.HTML {
	var path = IssuePath(i.Title.Slug, i.Slug)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, i.RawDate))
}

// WorkflowPath returns the path to perform a workflow action against this issue
func (i *Issue) WorkflowPath(action string) string {
	return IssueWorkflowPath(i.Title.Slug, i.Slug, action)
}

// ScanPDFImageDPIs runs through each PDF in the issue and checks it for DPI
// validity.  The results are cached to avoid this very costly process running
// too many times, and this should only be called when a user is viewing a
// single issue.  Running it on all issues' files for every scan could be
// disastrous.
func (i *Issue) ScanPDFImageDPIs() {
	for _, f := range i.Files {
		f.ValidateDPI()
	}
}

// File wraps a schema.File for web presentation
type File struct {
	*schema.File
	Issue  *Issue
	Slug   string
	Errors Errors

	// hasScannedPDFDPIs is used to avoid double-scanning the same file, since
	// the per-issue cost for this is fairly high
	hasScannedPDFDPIs bool
}

func (f *File) addError(err template.HTML) {
	f.Errors = append(f.Errors, err)
	f.Issue.AddChildError()
}

func (f *File) decorateErrors() {
	f.Errors = make(Errors, 0)
	for _, e := range f.Issue.Title.allErrors.IssueErrors[f.Issue.Issue] {
		if e.File != f.File {
			continue
		}
		f.addError(safeError(e.Error.Error()))
	}
}

// Link returns a link for this title
func (f *File) Link() template.HTML {
	var path = FilePath(f.Issue.Title.Slug, f.Issue.Slug, f.Slug)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, f.Slug))
}

// validDPI returns whether a file has a valid DPI for PDFs in scanned issues
func (f *File) validDPI() error {
	var maxDPI = float64(conf.ScannedPDFDPI) * 1.15
	var minDPI = float64(conf.ScannedPDFDPI) * 0.85

	var dpis = pdf.ImageDPIs(f.Location)
	if len(dpis) == 0 {
		return fmt.Errorf("contains no images or is invalid PDF")
	}

	for _, dpi := range dpis {
		if dpi.X > maxDPI || dpi.Y > maxDPI || dpi.X < minDPI || dpi.Y < minDPI {
			return fmt.Errorf("has an image with a bad DPI (%g x %g; expected DPI %d)", dpi.X, dpi.Y, conf.ScannedPDFDPI)
		}
	}

	return nil
}

// ValidateDPI adds errors to the file if its embedded images' DPIs are not
// within 15% of the configured scanner DPI.  This does nothing if the file's
// issue isn't scanned or if the file isn't a PDF.
func (f *File) ValidateDPI() {
	if f.Issue.Title.Type != TitleTypeScanned {
		return
	}
	if strings.ToUpper(filepath.Ext(f.Name)) != ".PDF" {
		return
	}
	if f.hasScannedPDFDPIs {
		return
	}

	var err = f.validDPI()
	if err != nil {
		f.addError(safeError(err.Error()))
	}

	f.hasScannedPDFDPIs = true
}
