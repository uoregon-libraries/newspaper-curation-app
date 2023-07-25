CREATE TABLE `pipelines` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `created_at` DATETIME NOT NULL,
  `started_at` DATETIME,
  `completed_at` DATETIME,
  `name` TINYTEXT NOT NULL,
  `object_type` TINYTEXT,
  `object_id` INT(11),
  `description` TEXT NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

CREATE INDEX pipelines_name ON `pipelines` (`name`(255));
CREATE INDEX pipelines_created_at ON `pipelines` (`created_at`);
CREATE INDEX pipelines_started_at ON `pipelines` (`started_at`);
CREATE INDEX pipelines_object_type ON `pipelines` (`object_type`(255));
CREATE INDEX pipelines_object_id ON `pipelines` (`object_id`);

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
