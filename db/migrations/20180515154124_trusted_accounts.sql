-- +migrate Up
ALTER TABLE profiles ADD COLUMN is_trusted BOOLEAN NOT NULL DEFAULT FALSE;

-- +migrate Down
ALTER TABLE profiles DROP COLUMN is_trusted;
