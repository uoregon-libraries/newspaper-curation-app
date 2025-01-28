-- +goose Up
ALTER TABLE `jobs` ADD COLUMN `entwine_id` BIGINT DEFAULT 0;

-- +goose Down
ALTER TABLE `jobs` DROP COLUMN `entwine_id`;
