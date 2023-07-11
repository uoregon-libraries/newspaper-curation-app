-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `audit_logs` MODIFY COLUMN `message` mediumtext COLLATE utf8_bin;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `audit_logs` MODIFY COLUMN `message` tinytext COLLATE utf8_bin;
