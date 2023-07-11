-- +goose Up
ALTER TABLE `issues` ADD `metadata_entered_at` DATETIME;
CREATE INDEX issue_metadata_entry ON `issues` (`metadata_entered_at`);

-- +goose Down
DROP INDEX issue_metadata_entry ON `issues`;
ALTER TABLE `issues` DROP COLUMN `metadata_entered_at`;
