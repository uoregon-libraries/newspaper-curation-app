package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Nerdmaster/magicsql"
	_ "github.com/go-sql-driver/mysql"
)

var Source *sql.DB
var DB *magicsql.DB

// Connect takes a bash config object, uses the database information contained
// therein, and stores both the source sql.DB and the wrapped magicsql.DB
func Connect(config map[string]string) error {
	var err error

	// DB string format: user:pass@tcp(127.0.0.1:3306)/db
	var connect = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", config["DB_USER"], config["DB_PASSWORD"],
		config["DB_HOST"], config["DB_PORT"], config["DB_DATABASE"])
	Source, err = sql.Open("mysql", connect)
	if err != nil {
		return err
	}
	Source.SetConnMaxLifetime(time.Second * 14400)
	DB = magicsql.Wrap(Source)

	return nil
}
