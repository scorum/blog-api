
-- +migrate Up
UPDATE posts_plagiarism
  SET uniqueness_percent=0.01
WHERE uniqueness_percent=0;



-- +migrate Down
