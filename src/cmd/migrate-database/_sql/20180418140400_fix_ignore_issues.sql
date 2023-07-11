-- +goose Up
ALTER TABLE `issues` MODIFY COLUMN `ignored` TINYINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE `issues` MODIFY COLUMN `ignored` TINYINT;
