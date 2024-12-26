-- +goose Up
ALTER TABLE `batches` ADD `full_name` TINYTEXT COLLATE utf8_bin;

-- +goose Down
ALTER TABLE `batches` DROP COLUMN `full_name`;
