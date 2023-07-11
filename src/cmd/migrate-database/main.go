// This is custom goose binary with sqlite3 support only.

package main

import (
	"embed"

	// We need to pull in mysql for the side-effect it offers us (allowing
	// "mysql" as a driver name), not the actual code it provides
	_ "github.com/go-sql-driver/mysql"

	"github.com/pressly/goose/v3"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/internal/logger"
)

//go:embed _sql
var migrations embed.FS

func getOpts() (*config.Config, []string) {
	var opts cli.BaseOptions
	var c = cli.New(&opts)
	c.AppendUsage(`Updates the database structure for NCA. Typically you'll ` +
		`only call this with the "up" command to migrate to the latest schema, ` +
		`but you may use any valid goose command -- at your own risk.`)
	var conf = c.GetConf()

	if len(c.Args) < 1 {
		c.UsageFail("You must specify a command (usually 'up')")
	}

	return conf, c.Args
}

func main() {
	var conf, args = getOpts()

	var db, err = goose.OpenDBWithDriver("mysql", conf.DatabaseConnect)
	if err != nil {
		logger.Fatalf("goose: failed to open DB: %v\n", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			logger.Fatalf("goose: failed to close DB: %v\n", err)
		}
	}()

	var command, subargs = args[0], args[1:]
	goose.SetBaseFS(migrations)
	if err := goose.Run(command, db, "_sql", subargs...); err != nil {
		logger.Fatalf("goose %v: %v", command, err)
	}
}
