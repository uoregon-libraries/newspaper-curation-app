-- +goose Up

-- Fix the "admin" role. People shouldn't have admin *and* other roles, but
-- they technically could, so we make this WHERE clause a bit ... weird.
UPDATE users SET roles = 'sysop' WHERE roles like '%admin%';

-- Deactivate any legacy "admin" users, then add the new sysop
UPDATE users SET deactivated = 1 WHERE login IN ('admin', 'sysadmin');
INSERT INTO users (`login`, `roles`) VALUES ('sysop', 'sysop');

-- +goose Down
DELETE FROM users WHERE login = 'sysop' AND roles = 'sysop';
UPDATE users SET deactivated = 0 WHERE login IN ('admin', 'sysadmin');
UPDATE users SET roles = 'admin' WHERE roles like 'sysop';
