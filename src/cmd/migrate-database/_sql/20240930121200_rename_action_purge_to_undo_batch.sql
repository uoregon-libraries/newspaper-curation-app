-- +goose Up
UPDATE actions SET action_type = 'undo-batch' WHERE action_type = 'purge-batch';

-- +goose Down
UPDATE actions SET action_type = 'purge-batch' WHERE action_type = 'undo-batch';
