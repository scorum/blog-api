-- +migrate Up
DELETE FROM media;
ALTER TABLE media ADD COLUMN meta jsonb NOT NULL;

-- +migrate Down
