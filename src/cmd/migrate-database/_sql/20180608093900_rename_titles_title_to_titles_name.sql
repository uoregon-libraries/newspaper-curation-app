-- +goose Up
ALTER TABLE `titles` CHANGE `title` `name` tinytext COLLATE utf8_bin;

-- +goose Down
ALTER TABLE `titles` CHANGE `name` `title` tinytext COLLATE utf8_bin;
