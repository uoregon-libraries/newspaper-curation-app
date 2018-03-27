package uploadedissuehandler

import (
	"apperr"
	"db"
	"fmt"
	"html/template"
	"issuesearch"
	"jobs"
	"os"

	"path/filepath"
	"schema"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/fileutil"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/pdf"
)

// DaysIssueConsideredDangerous is how long we require an issue to be untouched
// prior to anybody queueing it
const DaysIssueConsideredDangerous = 2

// DaysIssueConsideredNew is how long we warn users that the issue is new - but
// it can be queued before that warning goes away so long as
// DaysIssueConsideredDangerous has elapsed
const DaysIssueConsideredNew = 14

// An HTMLError implements apperr.Error but is meant to be displayed raw
// instead of escaped
type HTMLError struct {
	*apperr.BaseError
}

// HTMLify returns the errors joined together, with all non-HTMLError
// instances' messages escaped for use in HTML
func HTMLify(list apperr.List) template.HTML {
	var sList = make([]string, len(list))
	for i, e := range list {
		var msg = e.Message()
		if _, ok := e.(HTMLError); ok != true {
			msg = template.HTMLEscapeString(msg)
		}
		sList[i] = msg
	}
	return template.HTML(strings.Join(sList, "; "))
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
	issue.decorateExternalErrors()
	issue.scanModifiedTime()
	t.Issues = append(t.Issues, issue)
	t.IssueLookup[issue.Slug] = issue

	return issue
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
	Slug       string           // Short, URL-friendly identifier for an issue
	Title      *Title           // Title to which this issue belongs
	QueueInfo  template.HTML    // Informational message from the queue process, if any
	Files      []*File          // List of files
	FileLookup map[string]*File // Lookup for finding a File by its filename / slug
	Modified   time.Time        // When this issue's most recent file was modified
}

func (i *Issue) decorateFiles(fileList []*schema.File) {
	i.Files = make([]*File, 0)
	i.FileLookup = make(map[string]*File)
	for _, f := range fileList {
		i.appendSchemaFile(f)
	}
}

// scanModifiedTime forcibly pulls stats from the filesystem for the issue
// directory as well as every file to be sure we get real-time information if
// files are changed between cache refreshes.
func (i *Issue) scanModifiedTime() {
	var info, err = os.Stat(i.Location)
	if err != nil {
		logger.Errorf("Unable to stat %q: %s", i.Location, err)
		i.Modified = time.Now()
		return
	}
	i.Modified = info.ModTime()

	var files []os.FileInfo
	files, err = fileutil.Readdir(i.Location)
	if err != nil {
		logger.Errorf("Unable to read dir %q: %s", i.Location, err)
		i.Modified = time.Now()
		return
	}

	for _, file := range files {
		var mod = file.ModTime()
		if i.Modified.Before(mod) {
			i.Modified = mod
		}
	}
}

func (i *Issue) appendSchemaFile(f *schema.File) {
	var slug = filepath.Base(f.Location)
	var pdf = &File{File: f, Slug: slug, Issue: i}
	i.Files = append(i.Files, pdf)
	i.FileLookup[pdf.Slug] = pdf
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
		i.AddError(&HTMLError{BaseError: &apperr.BaseError{ErrorString: errstr}})
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
	return time.Since(i.Modified) < time.Hour*24*DaysIssueConsideredNew
}

// IsDangerouslyNew on the other hand tells us if the issue is so new that
// we're not okay with manual queueing even with a warning, because it's just
// not safe!
func (i *Issue) IsDangerouslyNew() bool {
	return time.Since(i.Modified) < time.Hour*24*DaysIssueConsideredDangerous
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
	Issue *Issue
	Slug  string

	// hasScannedPDFDPIs is used to avoid double-scanning the same file, since
	// the per-issue cost for this is fairly high
	hasScannedPDFDPIs bool
}

// Link returns a link for this title
func (f *File) Link() template.HTML {
	var path = FilePath(f.Issue.Title.Slug, f.Issue.Slug, f.Slug)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, f.Slug))
}

// validDPI returns whether a file has a valid DPI for PDFs in scanned issues
func (f *File) validDPI() apperr.Error {
	var maxDPI = float64(conf.ScannedPDFDPI) * 1.15
	var minDPI = float64(conf.ScannedPDFDPI) * 0.85

	var dpis = pdf.ImageDPIs(f.Location)
	if len(dpis) == 0 {
		return apperr.Errorf("contains no images or is invalid PDF")
	}

	for _, dpi := range dpis {
		if dpi.X > maxDPI || dpi.Y > maxDPI || dpi.X < minDPI || dpi.Y < minDPI {
			return apperr.Errorf("has an image with a bad DPI (%g x %g; expected DPI %d)", dpi.X, dpi.Y, conf.ScannedPDFDPI)
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
		f.AddError(err)
	}

	f.hasScannedPDFDPIs = true
}
