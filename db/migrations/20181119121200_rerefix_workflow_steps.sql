-- +goose Up
ALTER TABLE `issues` MODIFY `workflow_step` TINYTEXT COLLATE utf8_bin DEFAULT 'AwaitingProcessing';

-- +goose Down
ALTER TABLE `issues` MODIFY `workflow_step` ENUM(
  'AwaitingProcessing',
  'AwaitingPageReview',
  'ReadyForMetadataEntry',
  'AwaitingMetadataReview',
  'ReadyForMETSXML',
  'ReadyForBatching',
  'InProduction'
) DEFAULT 'AwaitingProcessing';
