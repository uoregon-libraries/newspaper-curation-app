package main

import (
	"bytes"
	"net/http"
	"path"

	"github.com/uoregon-libraries/gopkg/session"
	"github.com/uoregon-libraries/gopkg/tmpl"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

// responder wraps the server, request, and response writer to simplify common
// operations which need this data.  A responder will automatically render our
// 500 page if any errors occur, bypassing other application logic.
type responder struct {
	w      http.ResponseWriter
	req    *http.Request
	sess   *session.Session
	err    error
	server *srv
}

// router holds the server context and handler, and implements http.Handler,
// generating a request-wrapping responder for each incoming request to the
// router
type router struct {
	server  *srv
	handler func(r *responder)
}

// route returns a router for handling HTTP requests
func (s *srv) route(handler func(r *responder)) *router {
	return &router{server: s, handler: handler}
}

// ServeHTTP implements http.Handler so a responder can act as an arbitrary
// request handler
func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var sess, err = store.Session(w, req)
	if err != nil {
		sess, err = store.NewSession(w, req)
	}
	var response = &responder{w: w, req: req, sess: sess, server: r.server}
	if err != nil {
		r.server.logger.Warnf("Unable to instantiate session: %s", err)
	}

	r.handler(response)
}

func (r *responder) redirectSubpath(subpath string, code int) {
	http.Redirect(r.w, r.req, path.Join(r.server.webroot.Path, subpath), code)
}

func (r *responder) ajaxError(msg string, code int) {
	r.w.WriteHeader(code)
	r.w.Write([]byte(msg))
}

func (r *responder) error(msg string, code int) {
	r.w.WriteHeader(code)
	r.server.empty.Execute(r.w, map[string]string{"Alert": msg})
}

func (r *responder) internalServerError() {
	r.error("Internal server error.  Please try again, and contact support if the problem persists", 500)
}

func (r *responder) render(t *tmpl.Template, data map[string]interface{}) {
	var b = new(bytes.Buffer)

	var sessAlert = r.sess.GetAlertFlash()
	var sessInfo = r.sess.GetInfoFlash()
	var sessUser = r.sess.GetString("user")
	if data == nil {
		data = make(map[string]interface{})
	}
	data["Alert"] = sessAlert
	data["Info"] = sessInfo
	data["User"] = sessUser

	if sessUser != "" {
		var t, err = models.FindTitle("sftp_user = ?", sessUser)
		if err != nil {
			r.server.logger.Errorf("Unable to look up sftp user %q: %s", sessUser, err)
			r.internalServerError()
			return
		}

		if t.ID == 0 {
			r.server.logger.Errorf("Authenticated user %q somehow has no title in the database", sessUser)
			r.internalServerError()
			return
		}

		data["Title"] = t
	}

	var err = t.Execute(b, data)
	if err != nil {
		r.server.logger.Errorf("Unable to render template %s: %s", t.Name, err)
		r.internalServerError()
		return
	}

	b.WriteTo(r.w)
}
