-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Workflow ownership
ALTER TABLE `issues` ADD `metadata_entry_user_id` INT(11);
ALTER TABLE `issues` ADD `reviewed_by_user_id` INT(11);
ALTER TABLE `issues` ADD `workflow_owner_id` INT(11);
ALTER TABLE `issues` ADD `workflow_owner_expires_at` DATETIME;
ALTER TABLE `issues` DROP COLUMN `reviewed_by`;

-- Fix workflow so we can actually easily get the "step"
ALTER TABLE `issues` DROP COLUMN `awaiting_page_review`;
ALTER TABLE `issues` DROP COLUMN `ready_for_metadata_entry`;
ALTER TABLE `issues` DROP COLUMN `awaiting_metadata_review`;
ALTER TABLE `issues` ADD `workflow_step` ENUM(
  'AwaitingPageReview',
  'ReadyForMetadataEntry',
  'AwaitingMetadataReview',
  'ReadyForBatching'
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

-- Workflow ownership reversal
ALTER TABLE `issues` DROP COLUMN `metadata_entry_user_id`;
ALTER TABLE `issues` DROP COLUMN `reviewed_by_user_id`;
ALTER TABLE `issues` DROP COLUMN `workflow_owner_id`;
ALTER TABLE `issues` DROP COLUMN `workflow_owner_expires_at`;
ALTER TABLE `issues` ADD `reviewed_by` TINYTEXT COLLATE utf8_bin;

-- Unfix workflow data
ALTER TABLE `issues` ADD `awaiting_page_review` TINYINT;
ALTER TABLE `issues` ADD `ready_for_metadata_entry` TINYINT;
ALTER TABLE `issues` ADD `awaiting_metadata_review` TINYINT;
ALTER TABLE `issues` DROP COLUMN `workflow_step`;
