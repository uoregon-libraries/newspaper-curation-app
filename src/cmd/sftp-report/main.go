package main

import (
	"bashconf"
	"database/sql"
	"fileutil"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"user"
	"webutil"

	"github.com/Nerdmaster/magicsql"
	_ "github.com/go-sql-driver/mysql"
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

	// DB string format: user:pass@tcp(127.0.0.1:3306)/db
	var sqldb *sql.DB
	var connect = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", c["DB_USER"], c["DB_PASSWORD"], c["DB_HOST"],
		c["DB_PORT"], c["DB_DATABASE"])
	sqldb, err = sql.Open("mysql", connect)
	if err != nil {
		log.Fatal("Unable to connect to the database: %s", err)
	}
	sqldb.SetConnMaxLifetime(time.Second * 14400)
	user.DB = magicsql.Wrap(sqldb)

	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "Missing required parameter, <template path>\n\n", SFTPPath)
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}
	webutil.Webroot = opts.Webroot
	initTemplates(args[0])
}

func makeRedirect(dest string, code int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, dest, code)
	})
}

// checkSFTPWorkflow is an alias for the privilege-checking handlerfunc wrapper
func checkSFTPWorkflow(h http.HandlerFunc) http.Handler {
	return mustHavePrivilege(user.FindPrivilege("sftp workflow"), h)
}

func startServer() {
	var r = mux.NewRouter()
	var hp = webutil.HomePath()
	var pp = webutil.PublisherPath("{publisher}")
	var ip = webutil.IssuePath("{publisher}", "{issue}")
	var pdfPath = webutil.PDFPath("{publisher}", "{issue}", "{filename}")

	// Make sure homepath/ isn't considered the canonical path
	r.Handle(hp+"/", makeRedirect(hp, http.StatusMovedPermanently))

	r.NewRoute().Path(hp).Handler(checkSFTPWorkflow(HomeHandler))
	r.NewRoute().Path(pp).Handler(checkSFTPWorkflow(PublisherHandler))
	r.NewRoute().Path(ip).Handler(checkSFTPWorkflow(IssueHandler))
	r.NewRoute().Path(pdfPath).Handler(checkSFTPWorkflow(PDFFileHandler))

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
