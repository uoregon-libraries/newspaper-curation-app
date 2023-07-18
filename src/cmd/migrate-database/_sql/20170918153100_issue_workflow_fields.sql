-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;

ALTER TABLE `issues` ADD `location` TINYTEXT COLLATE utf8_bin;
ALTER TABLE `issues` ADD `is_from_scanner` TINYINT;
ALTER TABLE `issues` ADD `awaiting_page_review` TINYINT;
ALTER TABLE `issues` ADD `has_derivatives` TINYINT;
ALTER TABLE `issues` ADD `ready_for_metadata_entry` TINYINT;
ALTER TABLE `issues` ADD `awaiting_metadata_review` TINYINT;
ALTER TABLE `issues` ADD `reviewed_by` TINYTEXT COLLATE utf8_bin;

/*!40101 SET character_set_client = @saved_cs_client */;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;

ALTER TABLE `issues` DROP COLUMN `location`;
ALTER TABLE `issues` DROP COLUMN `is_from_scanner`;
ALTER TABLE `issues` DROP COLUMN `awaiting_page_review`;
ALTER TABLE `issues` DROP COLUMN `has_derivatives`;
ALTER TABLE `issues` DROP COLUMN `ready_for_metadata_entry`;
ALTER TABLE `issues` DROP COLUMN `awaiting_metadata_review`;
ALTER TABLE `issues` DROP COLUMN `reviewed_by`;

/*!40101 SET character_set_client = @saved_cs_client */;
