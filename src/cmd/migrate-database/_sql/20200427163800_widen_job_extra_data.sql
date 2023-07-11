-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `jobs` MODIFY `extra_data` MEDIUMTEXT COLLATE utf8_bin;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `jobs` MODIFY `extra_data` TINYTEXT COLLATE utf8_bin;
