-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE `batches` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `marc_org_code` TINYTEXT NOT NULL COLLATE utf8_bin,
  `created_at` DATETIME,
  `name` TINYTEXT NOT NULL COLLATE utf8_bin,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

ALTER TABLE `issues` ADD `batch_id` INT(11) NOT NULL DEFAULT 0;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
DROP TABLE `batches`;
ALTER TABLE `issues` DROP COLUMN `batch_id`;
