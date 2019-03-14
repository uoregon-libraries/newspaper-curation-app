// upload-server is set up very differently from the core NCA "server"
// application.  We're attempting to make this piece of NCA a much more
// independent server from the rest of the project because it has to be able to
// run on its own system, not the main NCA system.  Since it's already
// necessary to separate it some, we're aiming for a simpler and clearer
// codebase, so... we're hoping this accomplishes that.
package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/jessevdk/go-flags"
	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/gopkg/middleware"
	"github.com/uoregon-libraries/newspaper-curation-app/src/db"
)

var opts struct {
	BindAddress string `short:"b" long:"bind-address" description:"Web server's bind address" default:":8080"`
	AppRoot     string `long:"approot" description:"Filesystem path to app root, e.g., /usr/local/nca" default:"."`
	Webroot     string `long:"webroot" description:"Web root, e.g., \"https://example.edu/publisher/\"" default:"http://localhost:8080/"`
	Debug       bool   `long:"debug" description:"Enables debug mode for testing different users"`
}

var l *logger.Logger
var dbconn string

func getConf() {
	var p = flags.NewParser(&opts, flags.HelpFlag|flags.PassDoubleDash)
	var _, err = p.Parse()

	if err != nil {
		var flagsErr, ok = err.(*flags.Error)
		if !ok || flagsErr.Type != flags.ErrHelp {
			fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		}
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	dbconn = os.Getenv("NCA_DBCONNECT")
	if dbconn == "" {
		fmt.Fprintln(os.Stderr, "Error: NCA_DBCONNECT environment variable must be set")
		fmt.Fprintln(os.Stderr, `(e.g., NCA_DBCONNECT="user:password@tcp(localhost:port)/databasename")`)
		fmt.Fprintln(os.Stderr)
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	err = db.Connect(dbconn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %s\n\n", err)
		p.WriteHelp(os.Stderr)
		os.Exit(1)
	}
}

func newsrv() *srv {
	var s = new(srv)
	s.middleware = middleware.NewApache()
	s.logger = l
	s.middleware.Logger = l
	s.approot = opts.AppRoot
	s.debug = opts.Debug
	s.bindAddress = opts.BindAddress

	// We can ignore the error here because the config magic already verified
	// that the URL was valid
	var u, _ = url.Parse(opts.Webroot)
	s.webroot = u

	s.setupTemplates(filepath.Join(s.approot, "templates", "public-upload"))
	s.routes()

	return s
}

func listen() error {
	var s = newsrv()
	return s.listen()
}

func main() {
	l = logger.Named("nca-upload-server", logger.Debug)

	getConf()
	if opts.Debug == true {
		l.Warnf("Debug mode has been enabled")
	}

	l.Infof("Listening for %q on %q", opts.Webroot, opts.BindAddress)
	var err = listen()
	if err != nil {
		l.Fatalf("Unable to start server: %s", err)
	}
}
