// Package dbi is responsible for the low-level interfacing objects for
// external resources (DB and Debug for the database, SFTP for the SFTPGo
// connection) as well as the connection functions.  dbi is isolated here to
// avoid stupid circular dependencies since it needs to be accessible from just
// about every package.
package dbi

import (
	"database/sql"
	"net/url"
	"time"

	"github.com/Nerdmaster/magicsql"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/sftpgo"

	// We need to pull in mysql for the side-effect it offers us (allowing
	// "mysql" as a driver name), not the actual code it provides
	_ "github.com/go-sql-driver/mysql"
)

// DB is meant as a global accessor to a long-living database connection
var DB *magicsql.DB

// SFTP is our global sftpgo connection
var SFTP *sftpgo.API

// Debug should be set to true if operations should be logged to stderr
var Debug bool

// DBConnect takes a connection string, opens the database, and stores both the
// source sql.DB and the wrapped magicsql.DB
func DBConnect(connect string) error {
	// DB string format: user:pass@tcp(127.0.0.1:3306)/db
	var source, err = sql.Open("mysql", connect)
	if err != nil {
		return err
	}
	source.SetConnMaxLifetime(time.Second * 14400)
	DB = magicsql.Wrap(source)

	return nil
}

// SFTPConnect sets up the global SFTPGo API instance and attempts to grab a token
func SFTPConnect(u *url.URL, user, pass string) error {
	SFTP = sftpgo.New(u, user, pass)
	var err = SFTP.GetToken()
	return err
}
