package sftphandler

import (
	"db"
	"fmt"
	"html/template"
	"issuefinder"
	"issuesearch"
	"jobs"

	"path/filepath"
	"schema"
	"sort"
	"strings"
	"time"

	"github.com/uoregon-libraries/gopkg/logger"
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

// Title wraps a schema.Title with some extra information for web presentation.
type Title struct {
	*schema.Title
	Slug        string
	allErrors   []*issuefinder.Error
	Errors      Errors
	ChildErrors int
	Issues      []*Issue
	IssueLookup map[string]*Issue
}

// decorateTitles iterates over the list of the searcher's titles and decorates
// each, then its issues, and the issues' files, to prepare for web display
func (s *SFTPSearcher) decorateTitles() {
	s.titles = make([]*Title, 0)
	s.titleLookup = make(map[string]*Title)
	for _, t := range s.searcher.Titles {
		s.appendSchemaTitle(t)
	}

	// We like titles sorted by name for presentation
	sort.Slice(s.titles, func(i, j int) bool {
		return strings.ToLower(s.titles[i].Name) < strings.ToLower(s.titles[j].Name)
	})
}

func (s *SFTPSearcher) appendSchemaTitle(t *schema.Title) {
	var title = &Title{Title: t, Slug: t.LCCN, allErrors: s.searcher.Errors.Errors}
	title.decorateIssues(t.Issues)
	title.decorateErrors()
	s.titles = append(s.titles, title)
	s.titleLookup[title.Slug] = title
}

func (t *Title) decorateIssues(issueList []*schema.Issue) {
	t.Issues = make([]*Issue, 0)
	t.IssueLookup = make(map[string]*Issue)
	for _, i := range issueList {
		var _, isInProcess = sftpSearcher.inProcessIssues.Load(i.Key())
		if !isInProcess {
			t.appendSchemaIssue(i)
		}
	}
}

func (t *Title) appendSchemaIssue(i *schema.Issue) *Issue {
	var issue = &Issue{Issue: i, Slug: i.DateString(), Title: t}
	issue.decorateFiles(i.Files)
	issue.decorateErrors()
	issue.decorateExternalErrors()
	t.Issues = append(t.Issues, issue)
	t.IssueLookup[issue.Slug] = issue

	return issue
}

func (t *Title) decorateErrors() {
	t.Errors = make(Errors, 0)
	for _, e := range t.allErrors {
		if e.Title != t.Title {
			continue
		}
		if e.Issue == nil && e.File == nil {
			t.Errors = append(t.Errors, safeError(e.Error.Error()))
		} else {
			t.ChildErrors++
		}
	}
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
	Slug        string          // Short, URL-friendly identifier for an issue
	Title       *Title          // Title to which this issue belongs
	QueueInfo   template.HTML   // Informational message from the queue process, if any
	Errors      Errors          // List of errors automatically identified for this issue
	ChildErrors int             // Count of child errors for use in the templates
	PDFs        []*PDF          // List of "PDFs" - which are actually any associated files in the sftp issue's dir
	PDFLookup   map[string]*PDF // Lookup for finding a PDF by its filename / slug
	Modified    time.Time       // When this issue's most recent file was modified
}

func (i *Issue) decorateFiles(fileList []*schema.File) {
	i.PDFs = make([]*PDF, 0)
	i.PDFLookup = make(map[string]*PDF)
	for _, f := range fileList {
		i.appendSchemaFile(f)
		if i.Modified.Before(f.ModTime) {
			i.Modified = f.ModTime
		}
	}
}

func (i *Issue) appendSchemaFile(f *schema.File) {
	var slug = filepath.Base(f.Location)
	var pdf = &PDF{File: f, Slug: slug, Issue: i}
	pdf.decorateErrors()
	i.PDFs = append(i.PDFs, pdf)
	i.PDFLookup[pdf.Slug] = pdf
}

func (i *Issue) decorateErrors() {
	i.Errors = make(Errors, 0)
	for _, e := range i.Title.allErrors {
		if e.Issue != i.Issue {
			continue
		}

		if e.File == nil {
			i.Errors = append(i.Errors, safeError(e.Error.Error()))
		} else {
			i.ChildErrors++
		}
	}
}

// decorateExternalErrors checks for external problems we don't detect when
// just scanning the issue directories and files
func (i *Issue) decorateExternalErrors() {
	i.decorateDupeErrors()
	i.decoratePriorJobLogs()
}

// decorateDupeErrors adds errors to the issue if we find the same key in the global watcher
//
// TODO: Check the database for dupes as well!
func (i *Issue) decorateDupeErrors() {
	var key, err = issuesearch.ParseSearchKey(i.Key())
	// This shouldn't be able to happen, but if it does the best we can do is log
	// it and skip dupe checking; better than panicking in the lookup below
	if err != nil {
		logger.Errorf("Invalid issue key %q", i.Key())
		return
	}

	var watcherIssues = watcher.LookupIssues(key)
	for _, wi := range watcherIssues {
		var namespace = watcher.IssueFinder().IssueNamespace[wi]
		if namespace == issuefinder.SFTPUpload {
			continue
		}

		var errstr = "likely duplicate of "
		switch namespace {
		case issuefinder.Website:
			errstr += fmt.Sprintf(`a live issue: <a href="%s">%s, %s</a>`,
				wi.Location[:len(wi.Location)-5], wi.Title.Name, wi.DateStringReadable())
		case issuefinder.ScanUpload:
			errstr += "a scanned issue waiting for processing"
		default:
			errstr += fmt.Sprintf("an unknown issue (location: %q)", wi.Location)
		}

		i.Errors = append(i.Errors, template.HTML(errstr))
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
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, i.Date.Format("2006-01-02")))
}

// WorkflowPath returns the path to perform a workflow action against this issue
func (i *Issue) WorkflowPath(action string) string {
	return IssueWorkflowPath(i.Title.Slug, i.Slug, action)
}

// PDF wraps a schema.File for web presentation
type PDF struct {
	*schema.File
	Issue  *Issue
	Slug   string
	Errors Errors
}

func (p *PDF) decorateErrors() {
	p.Errors = make(Errors, 0)
	for _, e := range p.Issue.Title.allErrors {
		if e.File != p.File {
			continue
		}

		p.Errors = append(p.Errors, safeError(e.Error.Error()))
	}
}

// Link returns a link for this title
func (p *PDF) Link() template.HTML {
	var path = PDFPath(p.Issue.Title.Slug, p.Issue.Slug, p.Slug)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, p.Slug))
}
