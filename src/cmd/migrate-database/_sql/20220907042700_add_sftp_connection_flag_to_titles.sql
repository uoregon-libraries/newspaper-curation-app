-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `titles` ADD `sftp_connected` TINYINT;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `titles` DROP COLUMN `sftp_connected`;
