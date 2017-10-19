-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `issues` ADD `metadata_entry_user_id` INT(11);
ALTER TABLE `issues` ADD `reviewed_by_user_id` INT(11);
ALTER TABLE `issues` ADD `workflow_owner_id` INT(11);
ALTER TABLE `issues` ADD `workflow_owner_expires_at` DATETIME;
ALTER TABLE `issues` DROP COLUMN `reviewed_by`;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `issues` DROP COLUMN `metadata_entry_user_id`;
ALTER TABLE `issues` DROP COLUMN `reviewed_by_user_id`;
ALTER TABLE `issues` DROP COLUMN `workflow_owner_id`;
ALTER TABLE `issues` DROP COLUMN `workflow_owner_expires_at`;
ALTER TABLE `issues` ADD `reviewed_by` TINYTEXT COLLATE utf8_bin;
