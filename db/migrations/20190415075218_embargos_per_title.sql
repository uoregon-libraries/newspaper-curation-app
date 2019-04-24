-- +goose Up
ALTER TABLE `titles` ADD `embargo_period` TINYTEXT DEFAULT "0";
-- We set embargoed titles' period to "1 year" to avoid trying to pull in the
-- settings file.  This means manual modification is necessary, unfortunately,
-- but using the settings file would mean a much more complex script.
UPDATE `titles` SET `embargo_period` = '1 year' WHERE `embargoed` = 1;
ALTER TABLE `titles` DROP COLUMN `embargoed`;

-- +goose Down
ALTER TABLE `titles` ADD `embargoed` TINYINT DEFAULT 0;
UPDATE `titles` SET `embargoed` = 1 WHERE `embargo_period` <> '';
ALTER TABLE `titles` DROP COLUMN `embargo_period`;
