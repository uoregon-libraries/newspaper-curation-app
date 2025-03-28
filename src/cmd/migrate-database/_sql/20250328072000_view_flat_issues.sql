-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- This view is not exhaustive: it's meant to get at the most useful data, so
-- while the issue fields are comprehensive, the title and batch fields are
-- what I *think* make sense for most situations where this view might be used.
CREATE VIEW flat_issues AS
  SELECT
      -- Issue
      i.id, i.marc_org_code, i.lccn, i.date, i.date_as_labeled, i.volume,
      i.issue, i.edition, i.edition_label, i.page_labels_csv, i.location,
      i.is_from_scanner, i.metadata_entry_user_id, i.reviewed_by_user_id,
      i.workflow_owner_id, i.workflow_owner_expires_at, i.workflow_step,
      i.rejected_by_user_id, i.human_name, i.metadata_approved_at,
      i.backup_location, i.batch_id, i.ignored, i.draft_comment,
      i.metadata_entered_at, i.page_count,

      -- Title
      t.id AS title_id, t.name AS title_name, t.rights AS
      title_rights_statement, t.valid_lccn, t.marc_title, t.marc_location,
      t.embargo_period AS title_embargo_period, t.lang_code3 AS title_lang,

      -- Batch
      b.created_at AS batch_created_at, b.name AS batch_name, b.status AS
      batch_status, b.location AS batch_location, b.went_live_at,
      b.archived_at, b.full_name AS batch_full_name

    FROM issues i
    JOIN titles t ON (i.lccn = t.lccn)
    JOIN batches b ON (i.batch_id = b.id);

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
DROP VIEW flat_issues;
