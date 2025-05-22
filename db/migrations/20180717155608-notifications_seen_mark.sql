
-- +migrate Up
ALTER TABLE notifications
  ADD COLUMN is_seen BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE notifications SET is_seen = TRUE;

-- +migrate Down
ALTER TABLE notifications
  DROP COLUMN is_seen;
