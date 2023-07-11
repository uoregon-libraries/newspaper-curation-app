-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `jobs` ADD `run_at` DATETIME;
ALTER TABLE `jobs` ADD `queue_job_id` INT(11) NOT NULL;
ALTER TABLE `jobs` ADD `next_workflow_step` ENUM(
  '',
  'AwaitingProcessing',
  'AwaitingPageReview',
  'ReadyForMetadataEntry',
  'AwaitingMetadataReview',
  'ReadyForMETSXML',
  'ReadyForBatching'
);


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `jobs` DROP COLUMN `run_at`;
ALTER TABLE `jobs` DROP COLUMN `next_workflow_step`;
ALTER TABLE `jobs` DROP COLUMN `queue_job_id`;
