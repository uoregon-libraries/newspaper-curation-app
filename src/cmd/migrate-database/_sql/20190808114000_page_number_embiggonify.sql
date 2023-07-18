-- +goose Up
ALTER TABLE `issues` MODIFY `page_labels_csv` MEDIUMTEXT COLLATE utf8_bin;

-- +goose Down
ALTER TABLE `issues` MODIFY `page_labels_csv` TINYTEXT COLLATE utf8_bin;
