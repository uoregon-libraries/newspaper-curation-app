package migrations

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

func init() {
	goose.AddMigration(upAddBatchPermaname, noop)
}

func upAddBatchPermaname(tx *sql.Tx) error {
	var stmt, err = tx.Prepare("UPDATE batches SET full_name = ? WHERE id = ? AND (full_name IS NULL OR full_name = '')")
	if err != nil {
		return fmt.Errorf("preparing SQL: %w", err)
	}

	var batches []*models.Batch
	batches, err = models.AllBatches()
	if err != nil {
		return fmt.Errorf("reading batches from db: %w", err)
	}

	for _, b := range batches {
		if b.FullName == "" {
			b.GenerateFullName()
			var r, err = stmt.Exec(b.FullName, b.ID)
			if err != nil {
				return fmt.Errorf("saving batch %q: %w", b.FullName, err)
			}
			var n int64
			n, err = r.RowsAffected()
			if err != nil {
				return fmt.Errorf("counting rows affected by batch naming: %w", err)
			}
			if n != 1 {
				return fmt.Errorf("saving batch %q: %d rows affected instead of 1", b.FullName, n)
			}
		}
	}

	return nil
}
