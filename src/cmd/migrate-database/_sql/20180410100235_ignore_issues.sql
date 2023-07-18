-- +goose Up
ALTER TABLE `issues` ADD `ignored` TINYINT;

-- +goose Down
ALTER TABLE `issues` DROP COLUMN `ignored`;
