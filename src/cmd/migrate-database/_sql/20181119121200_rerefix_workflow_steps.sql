-- +goose Up
ALTER TABLE `issues` MODIFY `workflow_step` TINYTEXT COLLATE utf8_bin;

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
