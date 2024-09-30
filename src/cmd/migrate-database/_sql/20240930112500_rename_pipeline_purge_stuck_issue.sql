-- +goose Up
UPDATE pipelines SET name = 'DeleteStuckIssue' WHERE name = 'PurgeStuckIssue';

-- +goose Down
UPDATE pipelines SET name = 'PurgeStuckIssue' WHERE name = 'DeleteStuckIssue';
