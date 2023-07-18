-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `titles` ADD `marc_title` TINYTEXT COLLATE utf8_bin;
ALTER TABLE `titles` ADD `marc_location` TINYTEXT COLLATE utf8_bin;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `titles` DROP COLUMN `marc_title`;
ALTER TABLE `titles` DROP COLUMN `marc_location`;
