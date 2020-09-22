-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `issues` CHANGE `master_backup_location` `backup_location` TINYTEXT COLLATE utf8_bin;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
ALTER TABLE `issues` CHANGE `backup_location` `master_backup_location` TINYTEXT COLLATE utf8_bin;
