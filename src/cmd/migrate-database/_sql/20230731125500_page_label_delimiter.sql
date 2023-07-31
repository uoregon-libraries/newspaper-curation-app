-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
UPDATE `issues` SET page_labels_csv = REPLACE(page_labels_csv, ",", "␟");

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
UPDATE `issues` SET page_labels_csv = REPLACE(page_labels_csv, "␟", ",");
