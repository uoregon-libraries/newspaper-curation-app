-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `issues` MODIFY `workflow_step` ENUM(
  'AwaitingProcessing',
  'AwaitingPageReview',
  'ReadyForMetadataEntry',
  'AwaitingMetadataReview',
  'ReadyForMETSXML',
  'ReadyForBatching'
) DEFAULT 'AwaitingProcessing';

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `issues` MODIFY `workflow_step` ENUM(
  'AwaitingPageReview',
  'ReadyForMetadataEntry',
  'AwaitingMetadataReview',
  'ReadyForBatching'
);
