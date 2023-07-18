package uploadedissuehandler

import (
	"fmt"
	"html/template"
	"path/filepath"
	"strings"

	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/schema"
	"github.com/uoregon-libraries/newspaper-curation-app/src/uploads"
)

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
	MOC         string
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
	var uIssue = uploads.New(i, watcher.Scanner, conf)
	var issue = &Issue{
		Issue: uIssue,
		Slug:  i.DateEdition(),
		Title: t,
	}
	issue.decorateFiles(uIssue.Files)
	issue.decoratePriorJobLogs()
	issue.ValidateFast()
	t.Issues = append(t.Issues, issue)
	t.IssueLookup[issue.Slug] = issue

	return issue
}

// Show returns true if the title has any issues or errors.  If there are no
// errors and no issues, there's no reason to display it.
func (t *Title) Show() bool {
	return len(t.Issues) > 0 || t.HasErrors()
}

// HasErrors reports true if this title has any errors - due to the way
// AddError works in the schema, this will report true if the title has an
// error *or* if one or more issues have errors
func (t *Title) HasErrors() bool {
	return t.Errors.Major().Len() > 0
}

// Link returns a link for this title
func (t *Title) Link() template.HTML {
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, TitlePath(t.Slug()), t.Name))
}

// Slug generates a URL for the title based on its type, marc org code, and LCCN
func (t *Title) Slug() string {
	var parts = []string{"", t.MOC, t.LCCN}
	switch t.Type {
	case TitleTypeBornDigital:
		parts[0] = "dig"
	case TitleTypeScanned:
		parts[0] = "scan"
	}

	return strings.Join(parts, "-")
}

// Issue wraps uploads.Issue for web presentation
type Issue struct {
	*uploads.Issue
	Slug       string           // Short, URL-friendly identifier for an issue
	Title      *Title           // Title to which this issue belongs
	QueueInfo  template.HTML    // Informational message from the queue process, if any
	Files      []*File          // List of files
	FileLookup map[string]*File // Lookup for finding a File by its filename / slug
}

func (i *Issue) decorateFiles(fileList []*uploads.File) {
	i.Files = make([]*File, 0)
	i.FileLookup = make(map[string]*File)
	for _, f := range fileList {
		var slug = filepath.Base(f.Location)
		var pdf = &File{File: f, Slug: slug, Issue: i}
		i.Files = append(i.Files, pdf)
		i.FileLookup[pdf.Slug] = pdf
	}
}

// decoratePriorJobLogs adds information to issues that have old failed jobs.
func (i *Issue) decoratePriorJobLogs() {
	var dbi, err = models.FindIssueByKey(i.Key())
	if err != nil {
		logger.Warnf("Unable to look up issue for decorating queue messages: %s", err)
		return
	}
	if dbi == nil {
		return
	}

	var dbJobs []*models.Job
	dbJobs, err = dbi.Jobs()
	if err != nil {
		logger.Errorf("Unable to look up jobs for issue id %d (%q): %s", dbi.ID, i.Key(), err)
		return
	}

	var subErrors []string
	for _, j := range dbJobs {
		// We only care to report on the failed jobs, as those haven't been requeued
		if j.Status != string(models.JobStatusFailed) {
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

// Link returns a link for this title
func (i *Issue) Link() template.HTML {
	var path = IssuePath(i.Title.Slug(), i.Slug)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, i.RawDate))
}

// WorkflowPath returns the path to perform a workflow action against this issue
func (i *Issue) WorkflowPath(action string) string {
	return IssueWorkflowPath(i.Title.Slug(), i.Slug, action)
}

// HasErrors reports true if this issue has any errors - due to the way
// AddError works in the schema, this will report true if the issue has an
// error *or* if one or more files have errors
func (i *Issue) HasErrors() bool {
	return i.Errors.Major().Len() > 0
}

// HasWarnings reports true if this issue has any warnings
func (i *Issue) HasWarnings() bool {
	return i.Errors.Minor().Len() > 0
}

// ChildErrors reports the number of files with errors
func (i *Issue) ChildErrors() (n int) {
	for _, f := range i.Files {
		if f.HasErrors() {
			n++
		}
	}

	return n
}

// File wraps a schema.File for web presentation
type File struct {
	*uploads.File
	Issue *Issue
	Slug  string
}

// Link returns a link for this title
func (f *File) Link() template.HTML {
	var path = FilePath(f.Issue.Title.Slug(), f.Issue.Slug, f.Slug)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, f.Slug))
}

// RelativePath returns the location on disk relative to the title's location
// on disk (primarily to help locate SFTP issues)
func (f *File) RelativePath() string {
	return strings.Replace(f.Location, f.Issue.Title.Location, "", 1)
}

// HasErrors reports true if this file has any errors
func (f *File) HasErrors() bool {
	return f.Errors.Major().Len() > 0
}
