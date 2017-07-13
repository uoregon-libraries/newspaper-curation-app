package sftphandler

import (
	"fmt"
	"html/template"
	"issuefinder"
	"path/filepath"
	"schema"
	"sort"
	"strings"
	"time"
)

// Errors wraps an array of error strings for nicer display
type Errors []string

func (e Errors) String() string {
	return strings.Join(e, "; ")
}

// Title wraps a schema.Title with some extra information for web presentation.
// This is probably going to be SFTP-specific for now, but eventually (soon)
// needs to be useful in other contexts.
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
		t.appendSchemaIssue(i)
	}
}

func (t *Title) appendSchemaIssue(i *schema.Issue) {
	var issue = &Issue{Issue: i, Slug: i.DateString(), Title: t}
	issue.decorateFiles(i.Files)
	issue.decorateErrors()
	t.Issues = append(t.Issues, issue)
	t.IssueLookup[issue.Slug] = issue
}

func (t *Title) decorateErrors() {
	t.Errors = make(Errors, 0)
	for _, e := range t.allErrors {
		if e.Title != t.Title {
			continue
		}
		if e.Issue == nil && e.File == nil {
			t.Errors = append(t.Errors, e.Error.Error())
		} else {
			t.ChildErrors++
		}
	}
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
	Slug        string
	Title       *Title
	Errors      Errors
	ChildErrors int
	PDFs        []*PDF
	PDFLookup   map[string]*PDF
	Modified    time.Time
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
			i.Errors = append(i.Errors, e.Error.Error())
		} else {
			i.ChildErrors++
		}
	}
}

// IsNew tells the presentation if the issue is fairly new, which can be
// important for some publishers who upload over several days
func (i *Issue) IsNew() bool {
	return time.Since(i.Modified) < time.Hour*24*14
}

// Link returns a link for this title
func (i *Issue) Link() template.HTML {
	var path = IssuePath(i.Title.Slug, i.Slug)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, i.Date.Format("2006-01-02")))
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

		p.Errors = append(p.Errors, e.Error.Error())
	}
}

// Link returns a link for this title
func (p *PDF) Link() template.HTML {
	var path = PDFPath(p.Issue.Title.Slug, p.Issue.Slug, p.Slug)
	return template.HTML(fmt.Sprintf(`<a href="%s">%s</a>`, path, p.Slug))
}