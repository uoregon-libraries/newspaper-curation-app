-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `issues` ADD `error` MEDIUMTEXT COLLATE utf8_bin NOT NULL DEFAULT '';

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `issues` DROP COLUMN `error`;
