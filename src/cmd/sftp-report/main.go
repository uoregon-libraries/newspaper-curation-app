package main

import (
	"config"
	"db"
	"fmt"
	"log"
	"net/http"
	"os"
	"user"
	"webutil"

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
	ReportExit     bool   `long:"report-and-exit" description:"Show a textual SFTP report and exit the app"`
	Webroot        string `long:"webroot" description:"The base path to the app if it isn't just '/'"`
	ParentWebroot  string `long:"parent-webroot" description:"The base path to the parent app" required:"true"`
	StaticFilePath string `long:"static-files" description:"Path on disk to static JS/CSS/images" required:"true"`
}

// DEBUG is only enabled via command-line and should be used very sparingly,
// such as for user-switching (though an actual user-switch permission would be
// way better)
var DEBUG bool

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
		DEBUG = true
	}

	initTemplates(args[0])
}

func makeRedirect(dest string, code int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, dest, code)
	})
}

// canViewSFTPReport is an alias for the privilege-checking handlerfunc wrapper
func canViewSFTPReport(h http.HandlerFunc) http.Handler {
	return mustHavePrivilege(user.FindPrivilege("sftp report"), h)
}

func startServer() {
	var r = mux.NewRouter()
	var hp = webutil.HomePath()
	var pp = webutil.PublisherPath("{publisher}")
	var ip = webutil.IssuePath("{publisher}", "{issue}")
	var pdfPath = webutil.PDFPath("{publisher}", "{issue}", "{filename}")

	// Make sure homepath/ isn't considered the canonical path
	r.Handle(hp+"/", makeRedirect(hp, http.StatusMovedPermanently))

	r.NewRoute().Path(hp).Handler(canViewSFTPReport(HomeHandler))
	r.NewRoute().Path(pp).Handler(canViewSFTPReport(PublisherHandler))
	r.NewRoute().Path(ip).Handler(canViewSFTPReport(IssueHandler))
	r.NewRoute().Path(pdfPath).Handler(canViewSFTPReport(PDFFileHandler))

	// The static handler doesn't check permissions.  Right now this is okay, as
	// what we serve isn't valuable beyond page layout, but this may warrant a
	// fileserver clone + rewrite.
	r.NewRoute().PathPrefix(hp).Handler(http.StripPrefix(hp, http.FileServer(http.Dir(opts.StaticFilePath))))

	http.Handle("/", nocache(logMiddleware(r)))

	var addr = fmt.Sprintf("%s:%d", opts.Bind, opts.Port)
	log.Printf("Listening on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Error starting listener: %s", err)
	}
}

func main() {
	getConf()
	if opts.ReportExit {
		textReportOut()
		os.Exit(0)
	}

	startServer()
}
