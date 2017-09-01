-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `login` tinytext COLLATE utf8_bin,
  `roles` text COLLATE utf8_bin,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8 COLLATE=utf8_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
INSERT INTO `users` VALUES(1, "admin", "admin");
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
DROP TABLE `users`;
