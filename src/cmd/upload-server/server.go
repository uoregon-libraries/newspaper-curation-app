package main

import (
	"net/http"
	"net/url"
	"path"

	"github.com/gorilla/mux"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/middleware"
	"github.com/uoregon-libraries/gopkg/tmpl"
	"github.com/uoregon-libraries/gopkg/webutil"
	"github.com/uoregon-libraries/newspaper-curation-app/src/version"
)

type srv struct {
	router      *mux.Router
	approot     string
	webroot     *url.URL
	debug       bool
	logger      *logger.Logger
	middleware  *middleware.Middleware
	bindAddress string
	layout      *tmpl.TRoot
	empty       *tmpl.Template
}

// setupTemplates sets up pre-parsed templates
func (s *srv) setupTemplates(templatePath string) {
	var templateFunctions = tmpl.FuncMap{
		"Version":  func() string { return version.Version },
		"Debug":    func() bool { return s.debug },
		"LoginURL": func() string { return path.Join(s.webroot.Path, "login") },
	}

	// Set up the layout and then our global templates
	s.layout = tmpl.Root("layout", templatePath)
	webutil.Webroot = s.webroot.Path
	s.layout.Funcs(webutil.FuncMap)
	s.layout.Funcs(templateFunctions)
	s.layout.MustReadPartials("layout.go.html")
	s.empty = s.layout.MustBuild("empty.go.html")
}

func (s *srv) listen() error {
	var m = http.NewServeMux()
	m.Handle("/", s.router)
	var server = &http.Server{Addr: s.bindAddress, Handler: m}
	return server.ListenAndServe()
}
