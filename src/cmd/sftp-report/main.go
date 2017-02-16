package main

import (
	"bashconf"
	"fileutil"
	"fmt"
	"log"
	"net/http"
	"os"
	"webutil"

	"github.com/gorilla/mux"
	"github.com/jessevdk/go-flags"
)

var opts struct {
	ConfigFile     string `short:"c" long:"config" description:"path to P2C config file" required:"true"`
	Port           int    `short:"p" long:"port" description:"port to listen for HTTP traffic" required:"true"`
	Bind           string `long:"bind" description:"Bind address, usually safe to leave blank"`
	ReportExit     bool   `long:"report-and-exit" description:"Show a textual SFTP report and exit the app"`
	Webroot        string `long:"webroot" description:"The base path to the app if it isn't just '/'"`
	StaticFilePath string `long:"static-files" description:"Path on disk to static JS/CSS/images" required:"true"`
}

// SFTPPath gets the configured path to the SFTP root where each publisher
// directory resides
var SFTPPath string

func getConf() {
	var p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	p.Usage = "[OPTIONS] <template path>"
	var args, err = p.Parse()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	var c bashconf.Config
	c, err = bashconf.ReadFile(opts.ConfigFile)
	if err != nil {
		log.Fatal("Error parsing config file: %s", err)
	}

	SFTPPath = c["MASTER_PDF_UPLOAD_PATH"]
	if !fileutil.IsDir(SFTPPath) {
		fmt.Fprintf(os.Stderr, "Error: Cannot access SFTP path %#v\n\n", SFTPPath)
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Missing required parameter, <template path>\n\n", SFTPPath)
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}
	webutil.Webroot = opts.Webroot
	initTemplates(args[0])
}

func startServer() {
	var r = mux.NewRouter()
	var hp = webutil.FullPath(webutil.HomePath) + "/"
	r.HandleFunc(hp, HomeHandler)
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
