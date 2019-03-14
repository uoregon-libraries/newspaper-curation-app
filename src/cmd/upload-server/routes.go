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

	var fileServer = http.FileServer(http.Dir(filepath.Join(s.approot, "static", "public")))
	var staticRouter = s.router.NewRoute().PathPrefix(path.Join(s.webroot.Path, "static")).Subrouter()
	staticRouter.Use(s.middleware.RequestStaticAssetLog)
	staticRouter.NewRoute().Handler(http.StripPrefix(path.Join(s.webroot.Path, "static"), fileServer))

	var appRouter = s.router.NewRoute().PathPrefix(s.webroot.Path).Subrouter()
	appRouter.Use(s.middleware.NoCache)
	appRouter.Use(s.middleware.RequestLog)
	appRouter.Path("").HandlerFunc(s.redirectSubpathHandler("login", http.StatusMovedPermanently))
	appRouter.Path("/").HandlerFunc(s.redirectSubpathHandler("login", http.StatusMovedPermanently))
	appRouter.Path("/login").Methods("get").HandlerFunc(s.loginFormHandler())
	appRouter.Path("/login").Methods("post").HandlerFunc(s.loginSubmitHandler())

	appRouter.NotFoundHandler = s.notFoundHandler()
}
