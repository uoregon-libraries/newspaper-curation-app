-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
INSERT INTO actions (created_at, object_type, object_id, action_type, user_id, message)
  SELECT '2020-07-31', 'issue', issues.id, 'unfixable-error', -1, issues.error
  FROM issues
  WHERE
    issues.error <> '' AND
    issues.ignored = 0 AND
    issues.workflow_step = 'UnfixableMetadataError';

ALTER TABLE issues DROP COLUMN `error`;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
ALTER TABLE `issues` ADD `error` TINYTEXT COLLATE utf8_bin;
DROP TABLE actions;
