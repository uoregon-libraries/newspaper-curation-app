-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `issues` ADD `draft_comment` TEXT COLLATE utf8_bin;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
ALTER TABLE issues DROP COLUMN `draft_comment`;
