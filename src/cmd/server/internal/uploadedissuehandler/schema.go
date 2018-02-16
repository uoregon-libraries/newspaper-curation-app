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
	allErrors   *issuefinder.ErrorList
	Errors      Errors
	TitleErrors int
	ChildErrors int
	TotalErrors int
	Issues      []*Issue
	IssueLookup map[string]*Issue
}

// decorateTitles iterates over the list of the searcher's titles and decorates
// each, then its issues, and the issues' files, to prepare for web display
func (s *Searcher) decorateTitles() {
	var nextTitles = make([]*Title, 0)
	var nextTitleLookup = make(map[string]*Title)
	for _, t := range s.scanner.Finder.Titles {
		var title = &Title{Title: t, Slug: t.LCCN, allErrors: s.scanner.Finder.Errors}
		title.decorateIssues(t.Issues)
		title.decorateErrors()
		nextTitles = append(s.titles, title)
		nextTitleLookup[title.Slug] = title
	}

	// We like titles sorted by name for presentation
	sort.Slice(s.titles, func(i, j int) bool {
		return strings.ToLower(s.titles[i].Name) < strings.ToLower(s.titles[j].Name)
	})

	s.Lock()
	s.titles = nextTitles
	s.titleLookup = nextTitleLookup
	s.Unlock()
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

func (t *Title) decorateErrors() {
	t.Errors = make(Errors, 0)
	for _, e := range t.allErrors.TitleErrors[t.Title] {
		if e.Issue == nil && e.File == nil {
			t.Errors = append(t.Errors, safeError(e.Error.Error()))
			t.TitleErrors++
			t.TotalErrors++
		} else {
			t.ChildErrors++
			t.TotalErrors++
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
	Slug        string           // Short, URL-friendly identifier for an issue
	Title       *Title           // Title to which this issue belongs
	QueueInfo   template.HTML    // Informational message from the queue process, if any
	Errors      Errors           // List of errors automatically identified for this issue
	ChildErrors int              // Count of child errors for use in the templates
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

func (i *Issue) decorateErrors() {
	i.Errors = make(Errors, 0)
	for _, e := range i.Title.allErrors.IssueErrors[i.Issue] {
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
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, i.RawDate))
}

// WorkflowPath returns the path to perform a workflow action against this issue
func (i *Issue) WorkflowPath(action string) string {
	return IssueWorkflowPath(i.Title.Slug, i.Slug, action)
}

// File wraps a schema.File for web presentation
type File struct {
	*schema.File
	Issue  *Issue
	Slug   string
	Errors Errors
}

func (f *File) decorateErrors() {
	f.Errors = make(Errors, 0)
	for _, e := range f.Issue.Title.allErrors.IssueErrors[f.Issue.Issue] {
		if e.File != f.File {
			continue
		}

		f.Errors = append(f.Errors, safeError(e.Error.Error()))
	}
}

// Link returns a link for this title
func (f *File) Link() template.HTML {
	var path = FilePath(f.Issue.Title.Slug, f.Issue.Slug, f.Slug)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, f.Slug))
}
