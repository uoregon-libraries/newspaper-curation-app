-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `users` ADD COLUMN `deactivated` TINYINT NOT NULL DEFAULT 0;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
ALTER TABLE `users` DROP COLUMN `deactivated`;
