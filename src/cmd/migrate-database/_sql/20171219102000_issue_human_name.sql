-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `issues` ADD `human_name` TINYTEXT COLLATE utf8_bin;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `issues` DROP COLUMN `human_name`;
