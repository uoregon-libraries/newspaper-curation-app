package responder

import (
	"cmd/server/internal/settings"
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

// CanViewSFTPReport is an alias for the privilege-checking handlerfunc wrapper
func CanViewSFTPReport(h http.HandlerFunc) http.Handler {
	return MustHavePrivilege(user.FindPrivilege("sftp report"), h)
}

// MustHavePrivilege denies access to pages if there's no logged-in user, or
// there is a user but the user isn't allowed to perform a particular action
func MustHavePrivilege(priv *user.Privilege, f http.HandlerFunc) http.Handler {
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
		resp.Render("insufficient-privileges")
	})
}
