package main

import (
	"net/http"
	"path"

	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
)

func (s *srv) notFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		s.middleware.Log(w, req, http.HandlerFunc(http.NotFound), s.logger.Debugf, "Unrouted request")
	}
}

func (s *srv) redirectSubpath(w http.ResponseWriter, req *http.Request, subpath string, code int) {
	http.Redirect(w, req, path.Join(s.webroot.Path, subpath), code)
}

func (s *srv) redirectSubpathHandler(subpath string, code int) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		s.redirectSubpath(w, req, subpath, code)
	}
}

func (s *srv) loginFormHandler() http.HandlerFunc {
	var t = s.layout.MustBuild("login.go.html")
	return func(w http.ResponseWriter, req *http.Request) {
		s.render(w, req, t, nil)
	}
}

func (s *srv) loginSubmitHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		var t *db.Title
		var err error

		var name, pass = req.FormValue("loginname"), req.FormValue("password")
		if s.debug {
			// Let's make sure it's *really* hard to leave debug on by accident
			s.logger.Warnf("Debug mode: not validating password for %q", name)
			t, err = db.FindTitle("sftp_user = ?", name)
		} else {
			t, err = db.FindTitle("sftp_user = ? AND sftp_pass = ?", name, pass)
		}
		if err != nil {
			s.logger.Errorf("Unable to query database for user and password: %s", err)
			s.internalServerError(w, req)
			return
		}

		if t.ID == 0 {
			s.logger.Infof("Invalid login attempt for %q", name)
			s.redirectSubpath(w, req, "login", http.StatusSeeOther)
			return
		}

		// TODO: Store user info in session
		s.logger.Infof("%q authenticated for title %#v", name, t)
		s.redirectSubpath(w, req, "upload", http.StatusSeeOther)
	}
}

func (s *srv) error(w http.ResponseWriter, req *http.Request, msg string, code int) {
	w.WriteHeader(code)
	s.empty.Execute(w, map[string]string{"Alert": msg})
}

func (s *srv) internalServerError(w http.ResponseWriter, req *http.Request) {
	s.error(w, req, "Internal server error.  Please try again, and contact support if the problem persists", 500)
}
