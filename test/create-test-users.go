//go:build ignore

// create-test-users.go deletes all existing users from the database, then
// iterates over all assignable roles and creates a single-role user for each
// of them

package main

import (
	"strings"

	"github.com/uoregon-libraries/gopkg/logger"
	"github.com/uoregon-libraries/newspaper-curation-app/src/cli"
	"github.com/uoregon-libraries/newspaper-curation-app/src/config"
	"github.com/uoregon-libraries/newspaper-curation-app/src/dbi"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
	"github.com/uoregon-libraries/newspaper-curation-app/src/privilege"
)

var conf *config.Config

var opts cli.BaseOptions
var l = logger.New(logger.Debug, false)

func getOpts() {
	var c = cli.New(&opts)
	conf = c.GetConf()

	var err = dbi.DBConnect(conf.DatabaseConnect)
	if err != nil {
		l.Fatalf("Error trying to connect to database: %s", err)
	}
}

func main() {
	getOpts()

	var op = dbi.DB.Operation()
	var result = op.Exec("DELETE FROM users")
	if result.Err() != nil {
		l.Fatalf("Error deleting existing users: %s", result.Err())
	}

	for _, role := range privilege.AssignableRoles {
		var u = &models.User{Login: strings.ToLower(strings.Replace(role.Name, " ", "", -1))}
		u.Grant(role)
		l.Debugf("Creating user named %q with role %q", u.Login, role.Name)
		var err = u.Save()
		if err != nil {
			l.Fatalf("Error saving user for role %#v: %s", role, err)
		}
	}
}
