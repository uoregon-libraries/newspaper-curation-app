-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `batches` ADD `need_staging_purge` TINYINT;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
ALTER TABLE `batches` DROP COLUMN `need_staging_purge`;
