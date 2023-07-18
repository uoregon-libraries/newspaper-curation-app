-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;

ALTER TABLE `issues` DROP COLUMN `location`;
ALTER TABLE `issues` DROP COLUMN `workflow_step`;
ALTER TABLE `issues` DROP COLUMN `needs_derivatives`;
ALTER TABLE `issues` DROP COLUMN `info`;
ALTER TABLE `issues` DROP COLUMN `error`;

CREATE TABLE `jobs` (
  `id`                INT(11) NOT NULL AUTO_INCREMENT,
  `created_at`        DATETIME,
  `next_attempt_at`   DATETIME,
  `job_type`          TINYTEXT COLLATE utf8_bin,   /* SFTP Queue, page split, etc. */
  `object_id`         INT(11),                     /* DB id of the job's primary object, if relevant */
  `location`          TINYTEXT COLLATE utf8_bin,   /* Location of job object if relevant */
  `status`            TINYTEXT COLLATE utf8_bin,   /* started, succeeded, failed, etc. */
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;
CREATE TRIGGER `jobs_created_at`
  BEFORE INSERT ON `jobs`
  FOR EACH ROW
  SET NEW.created_at = NOW(), NEW.next_attempt_at = NOW();

CREATE TABLE `job_logs` (
  `id`                INT(11) NOT NULL AUTO_INCREMENT,
  `job_id`            INT(11) NOT NULL,
  `created_at`        DATETIME,
  `log_level`         TINYTEXT COLLATE utf8_bin,
  `message`           MEDIUMTEXT COLLATE utf8_bin,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;
CREATE TRIGGER `job_logs_created_at`
  BEFORE INSERT ON `job_logs`
  FOR EACH ROW
  SET NEW.created_at = NOW();

/*!40101 SET character_set_client = @saved_cs_client */;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TRIGGER `job_logs_created_at`;
DROP TABLE `job_logs`;
DROP TRIGGER `jobs_created_at`;
DROP TABLE `jobs`;
ALTER TABLE `issues` ADD `error` MEDIUMTEXT COLLATE utf8_bin;
ALTER TABLE `issues` ADD `info` MEDIUMTEXT COLLATE utf8_bin;
ALTER TABLE `issues` ADD `needs_derivatives` TINYINT;
ALTER TABLE `issues` ADD `workflow_step` TINYINT NOT NULL;
ALTER TABLE `issues` ADD `location` TINYTEXT COLLATE utf8_bin;
