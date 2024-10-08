-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `batches` ADD `oni_agent_job_id` BIGINT;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
ALTER TABLE `batches` DROP COLUMN `oni_agent_job_id`;
