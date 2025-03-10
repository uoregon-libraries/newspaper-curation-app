-- +goose Up
-- This is a no-op: if we delete this migration, goose may have odd issues
-- because it'll know the migration was applied, but won't be able to match a
-- file.
SELECT 1;

-- +goose Down
SELECT 1;
