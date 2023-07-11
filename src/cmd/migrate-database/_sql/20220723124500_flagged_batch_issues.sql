-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE `batches_flagged_issues` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `created_at` DATETIME,
  `flagged_by_user_id` INT(11) NOT NULL,
  `batch_id` INT(11) NOT NULL,
  `issue_id` INT(11) NOT NULL,
  `reason` TEXT NOT NULL,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=1 DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

CREATE UNIQUE INDEX bfi_batch_issue ON `batches_flagged_issues` (`batch_id`, `issue_id`);

CREATE TRIGGER `batches_flagged_issues_created_at`
  BEFORE INSERT ON `batches_flagged_issues`
  FOR EACH ROW
  SET NEW.created_at = NOW();

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
DROP TRIGGER `batches_flagged_issues_created_at`;
DROP TABLE `batches_flagged_issues`;
