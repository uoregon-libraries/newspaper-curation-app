-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
INSERT INTO users (login, roles) VALUES('sysadmin', 'admin');

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back
DELETE FROM users WHERE login = 'sysadmin';
