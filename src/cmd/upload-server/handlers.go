package main

import (
	"net/http"

	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
)

func (s *srv) notFoundHandler() *responder {
	return s.respond(func(r *responder) {
		s.middleware.Log(r.w, r.req, http.HandlerFunc(http.NotFound), s.logger.Debugf, "Unrouted request")
	})
}

func (s *srv) redirectSubpathHandler(subpath string, code int) *responder {
	return s.respond(func(r *responder) {
		r.redirectSubpath(subpath, code)
	})
}

func (s *srv) loginFormHandler() http.Handler {
	var t = s.layout.MustBuild("login.go.html")
	return s.respond(func(r *responder) {
		r.render(t, nil)
	})
}

func (s *srv) loginSubmitHandler() http.Handler {
	return s.respond(func(r *responder) {
		var t *db.Title
		var err error

		var name, pass = r.req.FormValue("loginname"), r.req.FormValue("password")
		if r.server.debug {
			// Let's make sure it's *really* hard to leave debug on by accident
			r.server.logger.Warnf("Debug mode: not validating password for %q", name)
			t, err = db.FindTitle("sftp_user = ?", name)
		} else {
			t, err = db.FindTitle("sftp_user = ? AND sftp_pass = ?", name, pass)
		}
		if err != nil {
			r.server.logger.Errorf("Unable to query database for user and password: %s", err)
			r.internalServerError()
			return
		}

		if t.ID == 0 {
			r.server.logger.Infof("Invalid login attempt for %q", name)
			r.sess.SetAlertFlash("Invalid login: username or password are incorrect")
			r.redirectSubpath("login", http.StatusSeeOther)
			return
		}

		r.sess.SetString("user", name)
		r.server.logger.Infof("%q authenticated for title %#v", name, t)
		r.redirectSubpath("upload", http.StatusSeeOther)
	})
}
