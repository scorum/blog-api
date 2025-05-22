
-- +migrate Up
CREATE TABLE deleted_posts(
  account TEXT NOT NULL REFERENCES profiles(account),
  permlink TEXT NOT NULL,
  created_at TIMESTAMP DEFAULT now(),
  PRIMARY KEY(account, permlink)
);

-- +migrate Down
DROP TABLE deleted_posts;