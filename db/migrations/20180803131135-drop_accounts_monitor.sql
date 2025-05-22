
-- +migrate Up
DROP TABLE IF EXISTS accounts_monitor;
ALTER TABLE posts RENAME TO comments;

-- +migrate Down
