-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Metadata review rejection
ALTER TABLE `issues` ADD `rejection_notes` TINYTEXT COLLATE utf8_bin;
ALTER TABLE `issues` ADD `rejected_by_user_id` INT(11);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

ALTER TABLE `issues` DROP COLUMN `rejection_notes`;
ALTER TABLE `issues` DROP COLUMN `rejected_by_user_id`;
