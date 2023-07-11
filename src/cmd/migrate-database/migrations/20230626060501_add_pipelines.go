package migrations

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
)

//go:embed "add_pipelines.sql"
var sqldata []byte

func init() {
	goose.AddMigration(upAddPipelines, downAddPipelines)
}

func runStatements(tx *sql.Tx, stmts []string) error {
	var err error
	for _, stmt := range stmts {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		_, err = tx.Exec(stmt)
		if err != nil {
			return err
		}
	}

	return nil
}

func upAddPipelines(tx *sql.Tx) error {
	// TODO: check for non-closed jobs and exit if any

	// Read in and process the pipelines sql
	var statements = strings.Split(string(sqldata), ";")
	return runStatements(tx, statements)
}

func downAddPipelines(_ *sql.Tx) error {
	return fmt.Errorf("Cannot migrate down from the Pipeline feature")
}
