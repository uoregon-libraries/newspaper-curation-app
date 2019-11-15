-- +goose Up
ALTER TABLE `mocs` ADD COLUMN `name` MEDIUMTEXT COLLATE utf8_bin;

-- +goose Down
ALTER TABLE `mocs` DROP COLUMN `name`;
