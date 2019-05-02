package main

import (
	"net/http"
	"path"
	"path/filepath"

	"github.com/gorilla/mux"
)

// routes defines all routing for the web server
func (s *srv) routes() {
	s.router = mux.NewRouter()

	var staticPath = path.Join(s.webroot.Path, "static")
	if staticPath[0] != '/' {
		staticPath = "/" + staticPath
	}

	var fileServer = http.FileServer(http.Dir(filepath.Join(s.approot, "static", "public")))
	var staticRouter = s.router.NewRoute().PathPrefix(staticPath).Subrouter()
	staticRouter.Use(s.middleware.RequestStaticAssetLog)
	staticRouter.NewRoute().Handler(http.StripPrefix(staticPath, fileServer))

	var appRouter = s.router.NewRoute().PathPrefix(s.webroot.Path).Subrouter()
	appRouter.Use(s.middleware.NoCache)
	appRouter.Use(s.middleware.RequestLog)
	appRouter.Path("").Handler(s.redirectSubpathHandler("login", http.StatusMovedPermanently))
	appRouter.Path("/").Handler(s.redirectSubpathHandler("login", http.StatusMovedPermanently))

	// TODO: Add redirect middleware if already logged in
	appRouter.Path("/login").Methods("get").Handler(s.loginFormHandler())
	appRouter.Path("/login").Methods("post").Handler(s.loginSubmitHandler())

	// TODO: Add login-required middleware
	appRouter.Path("/upload").Handler(s.uploadFormHandler())
	appRouter.Path("/upload/ajax").Handler(s.uploadAJAXReceiver())

	appRouter.NotFoundHandler = s.notFoundHandler()
}
