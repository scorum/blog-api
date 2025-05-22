-- +migrate Up
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION update_category(_domain DOMAIN, _label TEXT, _order INTEGER, _localization_key TEXT)
  RETURNS void AS $$
DECLARE
  previous_order INTEGER;
BEGIN
  SELECT "order" INTO STRICT previous_order FROM categories WHERE domain = _domain AND label = _label;

  if previous_order != _order THEN
    IF previous_order > _order THEN
      UPDATE categories SET "order" = ("order" + 1) WHERE domain = _domain AND "order" >= _order AND "order" < previous_order;
    ELSE
      UPDATE categories SET "order" = ("order" - 1) WHERE domain = _domain AND "order" <= _order AND "order" > previous_order;
    END IF;
  END IF;

  UPDATE categories SET "order" = _order, localization_key = _localization_key WHERE domain = _domain AND label = _label;
END;
$$ LANGUAGE 'plpgsql';
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION remove_category(_domain DOMAIN, _label TEXT)
  RETURNS void AS $$
DECLARE
  previous_order INTEGER;
BEGIN
  SELECT "order" INTO STRICT previous_order FROM categories WHERE domain = _domain AND label = _label;
  UPDATE categories SET "order" = ("order" - 1) WHERE domain = _domain AND "order" > previous_order;
  DELETE FROM categories WHERE domain = _domain AND label = _label;
END;
$$ LANGUAGE 'plpgsql';
-- +migrate StatementEnd

-- +migrate Down
DROP FUNCTION update_category(DOMAIN, TEXT, INTEGER, TEXT);
DROP FUNCTION remove_category(DOMAIN, TEXT);


