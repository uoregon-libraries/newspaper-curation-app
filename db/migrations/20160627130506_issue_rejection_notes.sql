-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `issue_rejection_notes` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `uuid` tinytext COLLATE utf8_bin,
  `notes` mediumtext COLLATE utf8_bin,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8 COLLATE=utf8_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE `issue_rejection_notes`;
