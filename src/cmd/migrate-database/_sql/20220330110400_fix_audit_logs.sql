-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
UPDATE audit_logs SET action = 'queue' WHERE action = '';
UPDATE audit_logs SET action = 'save-title' WHERE action = '';
UPDATE audit_logs SET action = 'validate-title' WHERE action = '';
UPDATE audit_logs SET action = 'create-moc' WHERE action = '';
UPDATE audit_logs SET action = 'update-moc' WHERE action = '';
UPDATE audit_logs SET action = 'delete-moc' WHERE action = '';
UPDATE audit_logs SET action = 'save-user' WHERE action = '';
UPDATE audit_logs SET action = 'deactivate-user' WHERE action = '';
UPDATE audit_logs SET action = 'claim' WHERE action = '	';
UPDATE audit_logs SET action = 'unclaim' WHERE action = '\n';
UPDATE audit_logs SET action = 'approve-metadata' WHERE action = '';
UPDATE audit_logs SET action = 'reject-metadata' WHERE action = '';
UPDATE audit_logs SET action = 'report-error' WHERE action = '';
UPDATE audit_logs SET action = 'undo-error-issue' WHERE action = '';
UPDATE audit_logs SET action = 'remove-error-issue' WHERE action = '';
UPDATE audit_logs SET action = 'queue-for-review' WHERE action = '';
UPDATE audit_logs SET action = 'autosave' WHERE action = '';
UPDATE audit_logs SET action = 'savedraft' WHERE action = '';
UPDATE audit_logs SET action = 'savequeue' WHERE action = '';

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back

-- There's no reason to migrate "down" from a data fix
SELECT 1;
