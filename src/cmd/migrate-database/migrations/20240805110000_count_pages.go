package migrations

import (
	"database/sql"
	"fmt"
	"log/slog"

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
	slog.Info("Counting all issues' pages")
	slog.Info("Loading all models. This may take a while...")
	var issues, err = models.Issues().AllowIgnored().Fetch()
	if err != nil {
		return fmt.Errorf("loading all issues: %w", err)
	}

	slog.Info("Load complete.")
	for i, issue := range issues {
		var mod = 1000
		if i < 1000 {
			mod = 100
		}

		if i%mod == 0 {
			slog.Info("Saving models...", "done", i, "remaining", len(issues)-i)
		}
		err = issue.SaveWithoutAction()
		if err != nil {
			return fmt.Errorf("saving issue %q: %w", issue.Key(), err)
		}
	}
	return nil
}

// downCountIssuePages is a no-op; page counts are deleted from the table anyway
func downCountIssuePages(_ *sql.Tx) error {
	return nil
}
