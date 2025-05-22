
-- +migrate Up
UPDATE posts_plagiarism
  SET uniqueness_percent = 1.0
WHERE status = 'legacy';

-- +migrate Down
