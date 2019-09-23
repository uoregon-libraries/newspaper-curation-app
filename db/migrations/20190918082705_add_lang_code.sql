-- +goose Up
ALTER TABLE `titles` ADD `lang_code3` TINYTEXT;

-- +goose Down
ALTER TABLE `titles` DROP COLUMN `lang_code3`;
