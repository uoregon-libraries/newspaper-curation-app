-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
DROP TABLE IF EXISTS `issues_processed`;
CREATE TABLE `issues_processed` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `issue_key` TINYTEXT NOT NULL COLLATE utf8_bin,
  `batch_name` TINYTEXT NOT NULL COLLATE utf8_bin,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

CREATE UNIQUE INDEX ip_issue_key ON `issues_processed` (`issue_key`(255));

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE `issues_processed`;
