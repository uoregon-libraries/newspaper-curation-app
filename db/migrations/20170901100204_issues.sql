-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `issues` (
  `id`                int(11) NOT NULL AUTO_INCREMENT,
	`marc_org_code`     TINYTEXT COLLATE utf8_bin,
	`lccn`              TINYTEXT COLLATE utf8_bin,
	`date`              TINYTEXT COLLATE utf8_bin,
	`date_as_labeled`   TINYTEXT COLLATE utf8_bin,
	`volume`            TINYTEXT COLLATE utf8_bin,
	`issue`             TINYTEXT COLLATE utf8_bin,
	`edition`           TINYINT NOT NULL,
	`edition_label`     TINYTEXT COLLATE utf8_bin,
	`page_labels_csv`   TINYTEXT COLLATE utf8_bin,

	`location`          TINYTEXT COLLATE utf8_bin,
	`workflow_step`     TINYINT NOT NULL,
	`needs_derivatives` TINYINT,
  `info`              MEDIUMTEXT COLLATE utf8_bin, /* Status message the end-user may need, but
                                                      which shouldn't prevent workflow actions */
  `error`             MEDIUMTEXT COLLATE utf8_bin, /* Error which prevents further action until
                                                      manual intervention occurs */

  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;
/*!40101 SET character_set_client = @saved_cs_client */;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE `issues`;
