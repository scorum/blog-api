-- +migrate Up
CREATE TABLE blockchain_monitor (
  block_num INT NOT NULL DEFAULT 0
);
INSERT INTO blockchain_monitor VALUES(0);

ALTER DOMAIN ACCOUNT DROP NOT NULL;

CREATE TABLE posts (
  permlink TEXT NOT NULL,
  author ACCOUNT  REFERENCES profiles(account) NOT NULL,
  body TEXT NOT NULL,
  title TEXT NOT NULL,
  json_metadata JSONB NOT NULL,
  parent_author ACCOUNT REFERENCES profiles(account) DEFAULT NULL,
  parent_permlink TEXT DEFAULT NULL,
  updated_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL,

  PRIMARY KEY (author, permlink)
);

CREATE TRIGGER post_updated_at
  BEFORE UPDATE ON posts
  FOR EACH ROW EXECUTE PROCEDURE updated_at_refresh();

-- +migrate Down
DROP TABLE posts;
ALTER TABLE blockchain_monitor RENAME TO accounts_monitor;
