-- +goose Up
DROP TRIGGER `batches_flagged_issues_created_at`;
CREATE TRIGGER `batches_flagged_issues_created_at`
  BEFORE INSERT ON `batches_flagged_issues`
  FOR EACH ROW
  SET NEW.created_at = UTC_TIMESTAMP();

DROP TRIGGER `job_logs_created_at`;
CREATE TRIGGER `job_logs_created_at`
  BEFORE INSERT ON `job_logs`
  FOR EACH ROW
  SET NEW.created_at = UTC_TIMESTAMP();

-- +goose Down
DROP TRIGGER `batches_flagged_issues_created_at`;
CREATE TRIGGER `batches_flagged_issues_created_at`
  BEFORE INSERT ON `batches_flagged_issues`
  FOR EACH ROW
  SET NEW.created_at = NOW();

DROP TRIGGER `job_logs_created_at`;
CREATE TRIGGER `job_logs_created_at`
  BEFORE INSERT ON `job_logs`
  FOR EACH ROW
  SET NEW.created_at = NOW();
