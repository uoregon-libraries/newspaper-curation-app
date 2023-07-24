-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `jobs` MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT UNIQUE;
ALTER TABLE `jobs` MODIFY COLUMN `object_id` BIGINT NOT NULL;
ALTER TABLE `job_logs` MODIFY COLUMN `id` BIGINT NOT NULL AUTO_INCREMENT UNIQUE;
ALTER TABLE `job_logs` MODIFY COLUMN `job_id` BIGINT NOT NULL;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
ALTER TABLE `job_logs` MODIFY COLUMN `job_id` int(11) NOT NULL;
ALTER TABLE `job_logs` MODIFY COLUMN `id` int(11) NOT NULL AUTO_INCREMENT;
ALTER TABLE `jobs` MODIFY COLUMN `object_id` int(11) NOT NULL;
ALTER TABLE `jobs` MODIFY COLUMN `id` int(11) NOT NULL AUTO_INCREMENT;
