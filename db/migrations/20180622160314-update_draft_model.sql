-- +migrate Up
ALTER TABLE drafts
  ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  ADD COLUMN updated_at TIMESTAMP DEFAULT NULL;

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION updated_at_refresh()
  RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE 'plpgsql';
-- +migrate StatementEnd

CREATE TRIGGER drafts_updated_at
  BEFORE UPDATE ON drafts
  FOR EACH ROW EXECUTE PROCEDURE updated_at_refresh();

-- +migrate Down
DROP TRIGGER drafts_updated_at ON drafts;
ALTER TABLE drafts DROP COLUMN updated_at;
ALTER TABLE drafts DROP COLUMN created_at;