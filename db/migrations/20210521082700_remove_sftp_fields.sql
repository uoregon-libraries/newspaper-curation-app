-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `titles` DROP COLUMN `sftp_dir`;
ALTER TABLE `titles` DROP COLUMN `sftp_pass`;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `titles` ADD `sftp_dir` TINYTEXT;
ALTER TABLE `titles` ADD `sftp_pass` TINYTEXT;
