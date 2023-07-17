package migrations

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
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
	var count int64
	var err = tx.QueryRow("SELECT COUNT(id) FROM jobs WHERE status NOT IN (?, ?)",
		models.JobStatusSuccessful, models.JobStatusFailedDone).Scan(&count)
	if err != nil {
		return fmt.Errorf("attempting to get count of unfinished jobs: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("cannot migrate to Pipeline feature with unfinished jobs")
	}

	// Read in and process the pipelines sql
	var statements = strings.Split(string(sqldata), ";")
	return runStatements(tx, statements)
}

func downAddPipelines(_ *sql.Tx) error {
	return fmt.Errorf("cannot migrate down from the Pipeline feature")
}
