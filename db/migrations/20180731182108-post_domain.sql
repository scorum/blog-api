
-- +migrate Up
ALTER TABLE posts
  ADD COLUMN domain "domain" NULL;


-- +migrate StatementBegin
CREATE FUNCTION asDomain(text) RETURNS "domain" LANGUAGE plpgsql IMMUTABLE AS $$
BEGIN
  RETURN CAST ($1 AS "domain");
EXCEPTION WHEN OTHERS THEN
  RETURN NULL;
END;$$;
-- +migrate StatementEnd

UPDATE posts
  SET "domain" = asDomain(replace(json_metadata->'domains'->>0, 'domain-', ''))
WHERE parent_author IS NULL;

CREATE INDEX posts_domain_idx ON posts("domain");


-- +migrate StatementBegin
DROP FUNCTION asDomain(text);
-- +migrate StatementEnd


-- +migrate Down
