-- +goose Up
ALTER TABLE `batches` ADD COLUMN `version` TINYINT DEFAULT 1;

-- +goose Down
ALTER TABLE `batches` DROP COLUMN `version`;
