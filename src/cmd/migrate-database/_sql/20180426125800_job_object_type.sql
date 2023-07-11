-- +goose Up
ALTER TABLE `jobs` ADD COLUMN `object_type` TINYTEXT COLLATE utf8_bin;

UPDATE `jobs` SET `object_type` = 'issue' WHERE `job_type` IN (
  'make_derivatives',
  'move_issue_to_workflow',
  'move_issue_to_page_review',
  'page_split',
  'build_mets',
  'move_master_files'
);

UPDATE `jobs` SET `object_type` = 'batch' WHERE `job_type` IN (
  'write_bagit_manifest',
  'move_batch_to_ready_location',
  'make_batch_xml',
  'create_batch_structure'
);

ALTER TABLE `jobs` MODIFY COLUMN `object_type` TINYTEXT COLLATE utf8_bin NOT NULL;

-- +goose Down
ALTER TABLE `jobs` DROP COLUMN `object_type`;
