-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;

DROP TRIGGER `jobs_created_at`;
ALTER TABLE `jobs` DROP COLUMN `next_attempt_at`;
ALTER TABLE `jobs` ADD `completed_at` DATETIME;
CREATE TRIGGER `jobs_created_at`
  BEFORE INSERT ON `jobs`
  FOR EACH ROW
  SET NEW.created_at = UTC_TIMESTAMP();

/*!40101 SET character_set_client = @saved_cs_client */;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TRIGGER `jobs_created_at`;
ALTER TABLE `jobs` DROP COLUMN `completed_at`;
ALTER TABLE `jobs` ADD `next_attempt_at` DATETIME;
CREATE TRIGGER `jobs_created_at`
  BEFORE INSERT ON `jobs`
  FOR EACH ROW
  SET NEW.created_at = NOW(), NEW.next_attempt_at = NOW();
