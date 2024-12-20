package migrations

import "database/sql"

// noop is for migrations that don't do anything, such as when a column is
// deleted in an SQL migration, so there's no need for a logic reversal.
func noop(_ *sql.Tx) error {
	return nil
}
