package sftphandler

import (
	"cmd/server/internal/responder"
	"fmt"
	"log"
	"net/http"
	"path"
	"web/tmpl"

	"github.com/gorilla/mux"
)

var (
	sftpSearcher *SFTPSearcher

	// workflowPath stores the directory where issues are moved when queued
	// for processing
	workflowPath string

	// basePath is the path to the main sftp page.  Subpages all start with this path.
	basePath string

	// Layout is the base template, cloned from the responder's layout, from
	// which all sftp pages are built
	Layout *tmpl.TRoot

	// HomeTmpl renders the main sftp reports page
	HomeTmpl *tmpl.Template

	// TitleTmpl renders the list of issues and a summary of errors for a given title
	TitleTmpl *tmpl.Template

	// IssueTmpl renders the list of PDFs and errors in a given issue
	IssueTmpl *tmpl.Template
)

// Setup sets up all the SFTP-specific routing rules and does any other
// init necessary for SFTP reports handling
func Setup(r *mux.Router, sftpWebPath, sftpDiskPath, sftpWorkflowPath string) {
	basePath = sftpWebPath
	workflowPath = sftpWorkflowPath
	var s = r.PathPrefix(basePath).Subrouter()
	s.Path("").Handler(responder.CanViewSFTPIssues(HomeHandler))
	s.Path("/{lccn}").Handler(responder.CanViewSFTPIssues(TitleHandler))
	s.Path("/{lccn}/{issue}").Handler(responder.CanViewSFTPIssues(IssueHandler))
	s.Path("/{lccn}/{issue}/workflow/{action}").Methods("POST").Handler(responder.CanWorkflowSFTPIssues(IssueWorkflowHandler))
	s.Path("/{lccn}/{issue}/{filename}").Handler(responder.CanViewSFTPIssues(PDFFileHandler))

	sftpSearcher = newSFTPSearcher(sftpDiskPath)
	Layout = responder.Layout.Clone()
	Layout.Path = path.Join(Layout.Path, "sftp")
	HomeTmpl = Layout.MustBuild("home.go.html")
	IssueTmpl = Layout.MustBuild("issue.go.html")
	TitleTmpl = Layout.MustBuild("title.go.html")
}

// LoadTitles takes a responder and attempts to load the title list
// into it.  If the list can't be loaded, an HTTP error is sent out and the
// return is false.
func LoadTitles(r *responder.Responder) bool {
	var titles, err = sftpSearcher.Titles()
	if err != nil {
		log.Printf("ERROR: Couldn't load titles in %s: %s", sftpSearcher.searcher.Location, err)
		http.Error(r.Writer, "Unable to load title list!", 500)
		return false
	}

	// TODO: Make responder act more as an embeddable type rather than the final renderer
	r.Vars.Data["Titles"] = titles
	return true
}

// findTitle attempts to load the title list, then find and return the
// title specified in the URL If no title is found (or loading
// title fails), nil is returned, and the caller should do nothing, as
// http headers / rendering is already done.
func findTitle(r *responder.Responder) *Title {
	if !LoadTitles(r) {
		return nil
	}
	var lccn = mux.Vars(r.Request)["lccn"]
	var title = sftpSearcher.TitleLookup(lccn)

	if title == nil {
		r.Vars.Alert = fmt.Sprintf("Unable to find title %#v", lccn)
		r.Render(responder.Empty)
		return nil
	}

	return title
}

// findIssue attempts to find the title specified in the URL and then the
// issue for that title, also specified in the URL.  If found, the issue is
// returned.  If not found, some kind of contextual error will be displayed to
// the end user and the caller should do nothing.
func findIssue(r *responder.Responder) *Issue {
	var title = findTitle(r)
	if title == nil {
		return nil
	}

	var issueDate = mux.Vars(r.Request)["issue"]
	var issue = title.IssueLookup[issueDate]

	if issue == nil {
		r.Vars.Alert = fmt.Sprintf("Unable to find issue %#v for title %#v", issueDate, title.Name)
		r.Render(responder.Empty)
		return nil
	}

	return issue
}

// HomeHandler spits out the title list
func HomeHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	if !LoadTitles(r) {
		return
	}

	r.Vars.Title = "SFTP Titles List"
	r.Render(HomeTmpl)
}

// TitleHandler prints a list of issues for a given title
func TitleHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var title = findTitle(r)
	if title == nil {
		return
	}

	r.Vars.Data["Title"] = title
	r.Vars.Title = "SFTP Issues for " + title.Name
	r.Render(TitleTmpl)
}

// IssueHandler prints a list of pages for a given issue
func IssueHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var issue = findIssue(r)
	if issue == nil {
		return
	}

	r.Vars.Data["Issue"] = issue
	r.Vars.Title = fmt.Sprintf("SFTP PDFs for %s, issue %s", issue.Title.Name, issue.Date.Format("2006-01-02"))
	r.Render(IssueTmpl)
}

// IssueWorkflowHandler handles SFTP workflow tasks: moving issues into the
// holding tank for derivative processing and renaming issues' folders to
// "*-error" with a user-defined message when issues are manually flagged
//
// All destructive operations run in the background, so a flag is set to tell
// the UI to consider the issue "pending workflow" until the next refresh
func IssueWorkflowHandler(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	var issue = findIssue(r)
	if issue == nil {
		return
	}

	var action = mux.Vars(r.Request)["action"]
	switch action {
	case "queue":
		queueIssueForProcessing(issue, workflowPath)
		http.SetCookie(w, &http.Cookie{
			Name:  "Alert",
			Value: fmt.Sprintf("Issue '%s' queued for processing", issue.Slug),
			Path:  "/",
		})
		http.Redirect(w, req, TitlePath(issue.Title.Slug), http.StatusFound)
	default:
		r.Vars.Alert = fmt.Sprintf("Invalid workflow action %#v", action)
		r.Render(responder.Empty)
	}
}
