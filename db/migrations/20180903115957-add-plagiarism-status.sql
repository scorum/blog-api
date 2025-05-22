
-- +migrate Up
CREATE TYPE plagiarism_status AS ENUM (
    'legacy',
    'pending',
    'checked',
    'failed'
  );


ALTER TABLE posts_plagiarism
  ADD COLUMN status plagiarism_status NOT NULL DEFAULT 'legacy';

ALTER TABLE posts_plagiarism
  DROP COLUMN failed_to_check;



-- +migrate Down
