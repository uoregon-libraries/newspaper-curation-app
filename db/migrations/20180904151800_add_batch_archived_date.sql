-- +goose Up
ALTER TABLE `batches` ADD `archived_at` DATETIME;

-- +goose Down
ALTER TABLE `batches` DROP COLUMN `archived_at`;
