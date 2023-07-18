-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `jobs` ADD COLUMN `extra_data` TINYTEXT COLLATE utf8_bin;
UPDATE `jobs` SET `extra_data` = `next_workflow_step`;
ALTER TABLE `jobs` DROP COLUMN `next_workflow_step`;


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `jobs` ADD `next_workflow_step` ENUM(
  '',
  'AwaitingProcessing',
  'AwaitingPageReview',
  'ReadyForMetadataEntry',
  'AwaitingMetadataReview',
  'ReadyForMETSXML',
  'ReadyForBatching'
);
UPDATE `jobs` SET `next_workflow_step` = `extra_data`;
ALTER TABLE `jobs` DROP COLUMN `extra_data`;
