-- +migrate Up
UPDATE drafts SET updated_at=created_at WHERE updated_at IS NULL;

ALTER TABLE drafts
  ALTER COLUMN updated_at SET NOT NULL,
  ALTER COLUMN updated_at SET DEFAULT CURRENT_TIMESTAMP;

-- +migrate Down
