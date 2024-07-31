-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `issues` ADD `page_count` INT DEFAULT 0;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
ALTER TABLE issues DROP COLUMN `page_count`;
