-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE `actions` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `created_at` DATETIME COLLATE utf8_bin,
  `object_type` TINYTEXT COLLATE utf8_bin,
  `object_id` INT(11) NOT NULL,
  `action_type` TINYTEXT COLLATE utf8_bin,
  `user_id` INT(11) NOT NULL,
  `message` TEXT COLLATE utf8_bin,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 COLLATE=utf8_bin;

INSERT INTO actions (created_at, object_type, object_id, action_type, user_id, message)
  SELECT '2020-05-01', 'issue', issues.id, 'metadata-rejection', issues.rejected_by_user_id, issues.rejection_notes
  FROM issues
  WHERE
    issues.rejection_notes <> '' AND
    issues.ignored = 0 AND
    issues.workflow_step <> 'ReadyForBatching';

UPDATE actions SET user_id = -1 WHERE user_id = 0;

ALTER TABLE issues DROP COLUMN `rejection_notes`;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
ALTER TABLE `issues` ADD `rejection_notes` TINYTEXT COLLATE utf8_bin;
DROP TABLE actions;
