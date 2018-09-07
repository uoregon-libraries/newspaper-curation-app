-- +goose Up
ALTER TABLE `batches` ADD `went_live_at` DATETIME;

-- +goose Down
ALTER TABLE `batches` DROP COLUMN `went_live_at`;
