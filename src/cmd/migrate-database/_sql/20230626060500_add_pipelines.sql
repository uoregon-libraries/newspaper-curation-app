-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE TABLE `pipelines` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `created_at` DATETIME NOT NULL,
  `description` TEXT NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

CREATE INDEX pipelines_created_at ON `pipelines` (`created_at`);

ALTER TABLE `jobs` ADD `pipeline_id` INT(11) DEFAULT -1;
ALTER TABLE `jobs` MODIFY COLUMN `pipeline_id` INT(11) NOT NULL;
ALTER TABLE `jobs` ADD `sequence` TINYINT DEFAULT -1;
ALTER TABLE `jobs` MODIFY COLUMN `sequence` TINYINT NOT NULL;
ALTER TABLE `jobs` DROP COLUMN `queue_job_id`;
CREATE INDEX jobs_pipeline_id ON `jobs` (`pipeline_id`);

CREATE TRIGGER `pipelines_created_at`
  BEFORE INSERT ON `pipelines`
  FOR EACH ROW
  SET NEW.created_at = UTC_TIMESTAMP();

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TRIGGER `pipelines_created_at`;
DROP INDEX jobs_pipeline_id ON `jobs`;
ALTER TABLE `jobs` ADD `queue_job_id` INT(11) NOT NULL;
ALTER TABLE `jobs` DROP COLUMN `sequence`;
ALTER TABLE `jobs` DROP COLUMN `pipeline_id`;
DROP INDEX pipelines_created_at ON `pipelines`;
DROP TABLE `pipelines`;
