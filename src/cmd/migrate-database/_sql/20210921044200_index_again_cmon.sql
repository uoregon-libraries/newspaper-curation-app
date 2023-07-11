-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE INDEX audit_logs_action ON `audit_logs` (`action`(255));

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
DROP INDEX audit_logs_action ON `audit_logs`;
