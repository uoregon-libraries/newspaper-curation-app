package db

import (
	"database/sql"
	"time"

	"github.com/Nerdmaster/magicsql"
	// We need to pull in mysql for the side-effect it offers us (allowing
	// "mysql" as a driver name), not the actual code it provides
	_ "github.com/go-sql-driver/mysql"
)

// DB is meant as a global accessor to a long-living database connection
var DB *magicsql.DB

// LastError is set any time an operation occurs which could result in an
// error, allowing code to flow freely with error handling only having to
// happen once instead of in multiple steps
var LastError error

// Debug should be set to true if operations should be logged to stderr
var Debug bool

// Connect takes a connection string, opens the database, and stores both the
// source sql.DB and the wrapped magicsql.DB
func Connect(connect string) error {
	// DB string format: user:pass@tcp(127.0.0.1:3306)/db
	var source, err = sql.Open("mysql", connect)
	if err != nil {
		return err
	}
	source.SetConnMaxLifetime(time.Second * 14400)
	DB = magicsql.Wrap(source)

	return nil
}
