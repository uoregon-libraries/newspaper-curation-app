package main

import (
	"cmd/server/internal/presenter"
	"fmt"
	"log"
	"net/http"
	"time"
	"user"

	"github.com/gorilla/mux"
)

func getUserLogin(w http.ResponseWriter, req *http.Request) string {
	var l string
	if DEBUG {
		l = req.URL.Query().Get("debuguser")
		if l == "" {
			var cookie, err = req.Cookie("debuguser")
			if err == nil {
				l = cookie.Value
			}
		}
		if l == "nil" {
			l = ""
			http.SetCookie(w, &http.Cookie{Name: "debuguser", Value: "", Expires: time.Time{}})
		} else {
			http.SetCookie(w, &http.Cookie{Name: "debuguser", Value: l})
		}
	}

	if l == "" {
		l = req.Header.Get("X-Remote-User")
	}

	return l
}

// Response generates a Responder with basic data all pages will need: request,
// response writer, and user
func Response(w http.ResponseWriter, req *http.Request) *Responder {
	var u = user.FindByLogin(getUserLogin(w, req))
	return &Responder{Writer: w, Request: req, Vars: &PageVars{User: u, Data: make(GenericVars)}}
}

// nocache is a Middleware function to send back no-cache header
func nocache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "max-age=0, must-revalidate")
		next.ServeHTTP(w, r)
	})
}

func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var u = getUserLogin(w, r)
		if u != "" {
			log.Printf("Request: [%s] %s", u, r.URL)
		} else {
			log.Printf("Request: [nil] %s", r.URL)
		}
		next.ServeHTTP(w, r)
	})
}

// mustHavePrivilege denies access to pages if there's no logged-in user, or
// there is a user but the user isn't allowed to perform a particular action
func mustHavePrivilege(priv *user.Privilege, f http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var u = user.FindByLogin(getUserLogin(w, r))
		var roles []*user.Role
		if u != nil {
			roles = u.Roles()
		}

		if priv.AllowedByAny(roles) {
			f(w, r)
			return
		}

		var resp = Response(w, r)
		resp.Vars.Title = "Insufficient Privileges"
		w.WriteHeader(http.StatusForbidden)
		resp.Render("insufficient-privileges")
	})
}

// LoadTitles takes a responder and attempts to load the title list
// into it.  If the list can't be loaded, an HTTP error is sent out and the
// return is false.
func LoadTitles(r *Responder) bool {
	var titles, err = sftpSearcher.Titles()
	if err != nil {
		log.Printf("ERROR: Couldn't load titles in %s: %s", Conf.MasterPDFUploadPath, err)
		http.Error(r.Writer, "Unable to load title list!", 500)
		return false
	}

	r.Vars.Titles = titles
	return true
}

// findTitle attempts to load the title list, then find and return the
// title specified in the URL If no title is found (or loading
// title fails), nil is returned, and the caller should do nothing, as
// http headers / rendering is already done.
func findTitle(r *Responder) *presenter.Title {
	if !LoadTitles(r) {
		return nil
	}
	var lccn = mux.Vars(r.Request)["lccn"]
	var title = sftpSearcher.TitleLookup(lccn)

	if title == nil {
		r.Vars.Alert = fmt.Sprintf("Unable to find title %#v", lccn)
		r.Render("empty")
		return nil
	}

	return title
}

// findIssue attempts to find the title specified in the URL and then the
// issue for that title, also specified in the URL.  If found, the issue is
// returned.  If not found, some kind of contextual error will be displayed to
// the end user and the caller should do nothing.
func findIssue(r *Responder) *presenter.Issue {
	var title = findTitle(r)
	if title == nil {
		return nil
	}

	var issueDate = mux.Vars(r.Request)["issue"]
	var issue = title.IssueLookup[issueDate]

	if issue == nil {
		r.Vars.Alert = fmt.Sprintf("Unable to find issue %#v for title %#v", issueDate, title.Name)
		r.Render("empty")
		return nil
	}

	return issue
}

// HomeHandler spits out the title list
func HomeHandler(w http.ResponseWriter, req *http.Request) {
	var r = Response(w, req)
	if !LoadTitles(r) {
		return
	}

	r.Vars.Title = "SFTP Titles List"
	r.Render("home")
}

// TitleHandler prints a list of issues for a given title
func TitleHandler(w http.ResponseWriter, req *http.Request) {
	var r = Response(w, req)
	var title = findTitle(r)
	if title == nil {
		return
	}

	r.Vars.Data["Title"] = title
	r.Vars.Title = "SFTP Issues for " + title.Name
	r.Render("title")
}

// IssueHandler prints a list of pages for a given issue
func IssueHandler(w http.ResponseWriter, req *http.Request) {
	var r = Response(w, req)
	var issue = findIssue(r)
	if issue == nil {
		return
	}

	r.Vars.Data["Issue"] = issue
	r.Vars.Title = fmt.Sprintf("SFTP PDFs for %s, issue %s", issue.Title.Name, issue.Date.Format("2006-01-02"))
	r.Render("issue")
}
