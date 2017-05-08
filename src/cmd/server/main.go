package main

import (
	"cmd/server/internal/findhandler"
	"cmd/server/internal/responder"
	"cmd/server/internal/settings"
	"cmd/server/internal/sftphandler"
	"config"
	"db"
	"fmt"
	"legacyfinder"
	"log"
	"net/http"
	"os"
	"path"
	"time"
	"user"
	"web/webutil"

	"github.com/gorilla/mux"
	"github.com/jessevdk/go-flags"
)

// Various command-line options live here, and yes, they're awful
//
// TODO:
// - Put more of this stuff into the central "bash" config
// - Migrate parent app in here entirely to get rid of some of the odd stuff
//   that needs ParentWebroot
var opts struct {
	ConfigFile     string `short:"c" long:"config" description:"path to P2C config file" required:"true"`
	Port           int    `short:"p" long:"port" description:"port to listen for HTTP traffic" required:"true"`
	Bind           string `long:"bind" description:"Bind address, usually safe to leave blank"`
	Debug          bool   `long:"debug" description:"Enables debug mode for testing different users"`
	ChronamRoot    string `long:"chronam-web-root" description:"Full URL to live site; e.g. http://oregonnews.uoregon.edu" required:"true"`
	Webroot        string `long:"webroot" description:"The base path to the app if it isn't just '/'"`
	ParentWebroot  string `long:"parent-webroot" description:"The base path to the parent app" required:"true"`
	StaticFilePath string `long:"static-files" description:"Path on disk to static JS/CSS/images" required:"true"`
}

// Conf stores the configuration data read from the legacy Python settings
var Conf *config.Config

func getConf() {
	var p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	p.Usage = "[OPTIONS] <template path>"
	var args, err = p.Parse()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	Conf, err = config.Parse(opts.ConfigFile)
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	err = db.Connect(Conf.DatabaseConnect)
	if err != nil {
		log.Fatalf("Error trying to connect to database: %s", err)
	}
	user.DB = db.DB

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Missing required parameter, <template path>\n\n")
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}
	webutil.Webroot = opts.Webroot
	webutil.ParentWebroot = opts.ParentWebroot

	if opts.Debug == true {
		log.Printf("WARNING: Debug mode has been enabled")
		settings.DEBUG = true
	}

	responder.InitRootTemplate(args[0])
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
	var fileServer = http.FileServer(http.Dir(opts.StaticFilePath))
	var staticPrefix = path.Join(hp, "static")
	r.NewRoute().PathPrefix(staticPrefix).Handler(http.StripPrefix(staticPrefix, fileServer))

	var watcher = legacyfinder.NewWatcher(Conf, opts.ChronamRoot)
	go watcher.Watch(5 * time.Minute)
	sftphandler.Setup(r, path.Join(hp, "sftp"), Conf.MasterPDFUploadPath)
	findhandler.Setup(r, path.Join(hp, "search-issues"), watcher)

	var waited, lastWaited int
	for watcher.IssueFinder().Issues == nil {
		if waited == 5 {
			log.Println("Waiting for initial issue scan to complete.  This can take " +
				"several minutes if the cache has not already been created.")
		} else if waited / 30 > lastWaited {
			log.Println("Still waiting...")
			lastWaited = waited/30
		}
		waited++
		time.Sleep(1 * time.Second)
	}

	http.Handle("/", nocache(logMiddleware(r)))

	var addr = fmt.Sprintf("%s:%d", opts.Bind, opts.Port)
	log.Printf("Listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Error starting listener: %s", err)
	}
}

func main() {
	getConf()
	startServer()
}
