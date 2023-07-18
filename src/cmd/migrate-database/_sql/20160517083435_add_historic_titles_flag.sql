-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `titles` ADD `is_historic` TINYINT NOT NULL DEFAULT 0;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `titles` DROP COLUMN `is_historic`;
