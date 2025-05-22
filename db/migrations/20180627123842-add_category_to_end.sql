-- +migrate Up
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION add_category(_domain DOMAIN, _label TEXT, _localization_key TEXT)
  RETURNS void AS $$
DECLARE
  last_order INTEGER;
BEGIN
  SELECT COALESCE(MAX("order"),0) INTO last_order FROM categories WHERE domain = _domain;
  INSERT INTO categories VALUES(_domain, _label, last_order + 1, _localization_key);
END;
$$ LANGUAGE 'plpgsql';
-- +migrate StatementEnd

-- +migrate Down
DROP FUNCTION add_category(DOMAIN, TEXT, TEXT);



