package migrations

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/uoregon-libraries/newspaper-curation-app/src/models"
)

func init() {
	goose.AddMigration(upCountIssuePages, downCountIssuePages)
}

// upCountIssuePages is an expensive little hack, but it's a one-time cost: we
// load each issue and re-save it so it uses the model's built-in logic to
// cache page counts
func upCountIssuePages(_ *sql.Tx) error {
	var issues, err = models.Issues().AllowIgnored().Fetch()
	if err != nil {
		return fmt.Errorf("loading all issues: %w", err)
	}
	for _, i := range issues {
		err = i.SaveWithoutAction()
		if err != nil {
			return fmt.Errorf("saving issue %q: %w", i.Key(), err)
		}
	}
	return nil
}

// downCountIssuePages is a no-op; page counts are deleted from the table anyway
func downCountIssuePages(_ *sql.Tx) error {
	return nil
}
