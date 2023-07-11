-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `issues` DROP COLUMN `has_derivatives`;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `issues` ADD `has_derivatives` TINYINT;
