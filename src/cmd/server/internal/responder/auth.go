package responder

import (
	"cmd/server/internal/settings"
	"logger"
	"net/http"
	"time"
	"user"
)

// GetUserLogin returns the Apache-auth user or the debuguser argument if
// settings.DEBUG is true
func GetUserLogin(w http.ResponseWriter, req *http.Request) string {
	var l string
	if settings.DEBUG {
		l = req.URL.Query().Get("debuguser")
		if l == "" {
			var cookie, err = req.Cookie("debuguser")
			if err == nil {
				l = cookie.Value
			}
		}
		if l == "nil" {
			l = ""
			http.SetCookie(w, &http.Cookie{Name: "debuguser", Value: "", Expires: time.Time{}, Path: "/"})
			logger.Debug(`Explicit request to clear "debuguser" cookie`)
		} else if l != "" {
			http.SetCookie(w, &http.Cookie{Name: "debuguser", Value: l, Path: "/"})
			logger.Debug(`Setting cookie: debuguser="%s"`, l)
		}
	}

	if l == "" {
		l = req.Header.Get("X-Remote-User")
	}

	return l
}

// CanViewSFTPIssues is an alias for the privilege-checking handlerfunc wrapper
func CanViewSFTPIssues(h http.HandlerFunc) http.Handler {
	return MustHavePrivilege("sftp report", h)
}

// CanWorkflowSFTPIssues is an alias for the privilege-checking handlerfunc
// wrapper, and tells us if a user is allowed to move SFTP issues forward,
// reject them, etc.
func CanWorkflowSFTPIssues(h http.HandlerFunc) http.Handler {
	return MustHavePrivilege("sftp workflow", h)
}

// CanSearchIssues is an alias for the privilege-checking handlerfunc wrapper
func CanSearchIssues(h http.HandlerFunc) http.Handler {
	return MustHavePrivilege("search workflow issues", h)
}

// MustHavePrivilege denies access to pages if there's no logged-in user, or
// there is a user but the user isn't allowed to perform a particular action
func MustHavePrivilege(privName string, f http.HandlerFunc) http.Handler {
	var priv = user.FindPrivilege(privName)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var u = user.FindByLogin(GetUserLogin(w, r))
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
		resp.Render(InsufficientPrivileges)
	})
}
