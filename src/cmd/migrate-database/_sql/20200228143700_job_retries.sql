-- +goose Up
ALTER TABLE `jobs` ADD COLUMN `retry_count` INT(11) COLLATE utf8_bin;

-- +goose Down
ALTER TABLE `jobs` DROP COLUMN `retry_count`;
