package main

import (
	"cmd/server/internal/issuefinderhandler"
	"cmd/server/internal/mochandler"
	"cmd/server/internal/responder"
	"cmd/server/internal/settings"
	"cmd/server/internal/titlehandler"
	"cmd/server/internal/uploadedissuehandler"
	"cmd/server/internal/userhandler"
	"cmd/server/internal/workflowhandler"
	"config"
	"db"
	"fmt"
	"issuewatcher"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"
	"web/webutil"

	"github.com/gorilla/mux"
	flags "github.com/jessevdk/go-flags"
	"github.com/uoregon-libraries/gopkg/logger"
)

var opts struct {
	ConfigFile string `short:"c" long:"config" description:"path to NCA config file" required:"true"`
	Debug      bool   `long:"debug" description:"Enables debug mode for testing different users"`
}

var conf *config.Config

func getConf() {
	var p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	var _, err = p.Parse()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	conf, err = config.Parse(opts.ConfigFile)
	if err != nil {
		logger.Fatalf("Config error: %s", err)
	}

	err = db.Connect(conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("Error trying to connect to database: %s", err)
	}

	// We can ignore the error here because the config magic already verified
	// that the URL was valid
	var u, _ = url.Parse(conf.Webroot)
	webutil.Webroot = u.Path
	webutil.WorkflowPath = conf.WorkflowPath
	webutil.IIIFBaseURL = conf.IIIFBaseURL

	if opts.Debug == true {
		logger.Warnf("Debug mode has been enabled")
		settings.DEBUG = true
	}

	responder.InitRootTemplate(filepath.Join(conf.AppRoot, "templates"))
}

func makeRedirect(dest string, code int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, dest, code)
	})
}

func startServer() {
	var r = mux.NewRouter()
	var hp = webutil.HomePath()

	// Make sure homepath/ isn't considered the canonical path
	r.Handle(hp+"/", makeRedirect(hp, http.StatusMovedPermanently))

	// The static handler doesn't check permissions.  Right now this is okay, as
	// what we serve isn't valuable beyond page layout, but this may warrant a
	// fileserver clone + rewrite.
	var fileServer = http.FileServer(http.Dir(filepath.Join(conf.AppRoot, "static")))
	var staticPrefix = path.Join(hp, "static")
	r.NewRoute().PathPrefix(staticPrefix).Handler(http.StripPrefix(staticPrefix, fileServer))

	var watcher = issuewatcher.New(conf)
	go watcher.Watch(5 * time.Minute)

	var waited, lastWaited int
	for watcher.Scanner.Finder.Issues == nil {
		if waited == 5 {
			logger.Infof("Waiting for initial issue scan to complete.  This can take " +
				"several minutes if the issues haven't been scanned in a while.  If this " +
				"is the first time scanning the live site, expect 10 minutes or more to " +
				"build the web JSON cache.")
		} else if waited/30 > lastWaited {
			logger.Infof("Still waiting...")
			lastWaited = waited / 30
		}
		waited++
		time.Sleep(1 * time.Second)
	}

	// Set up routing for various "sub-apps"
	uploadedissuehandler.Setup(r, path.Join(hp, "uploadedissues"), conf, watcher)
	workflowhandler.Setup(r, path.Join(hp, "workflow"), conf, watcher)
	issuefinderhandler.Setup(r, path.Join(hp, "find"), conf, watcher)
	mochandler.Setup(r, path.Join(hp, "mocs"), conf)
	userhandler.Setup(r, path.Join(hp, "users"), conf)
	titlehandler.Setup(r, path.Join(hp, "titles"), conf)

	// Any unknown paths get a semi-friendly 404
	r.NewRoute().PathPrefix("").HandlerFunc(notFound)

	http.Handle("/", nocache(logMiddleware(r)))

	logger.Infof("Listening on %s", conf.BindAddress)
	if err := http.ListenAndServe(conf.BindAddress, nil); err != nil {
		logger.Fatalf("Error starting listener: %s", err)
	}
}

func notFound(w http.ResponseWriter, req *http.Request) {
	var r = responder.Response(w, req)
	r.Error(http.StatusNotFound, "")
}

func main() {
	getConf()
	startServer()
}
