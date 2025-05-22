-- +migrate Up
CREATE TYPE content_type AS ENUM('image/jpeg', 'image/png', 'image/gif');
ALTER TABLE media ADD content_type content_type DEFAULT 'image/jpeg';

-- +migrate Down
ALTER TABLE media DROP content_type;
DROP TYPE content_type;