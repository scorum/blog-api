
-- +migrate Up
  CREATE TABLE posts_plagiarism (
    author ACCOUNT REFERENCES profiles(account) NOT NULL,
    permlink TEXT NOT NULL,
    last_check_at TIMESTAMP NOT NULL DEFAULT now(),
    uniqueness_percent REAL NOT NULL DEFAULT 1,
    urls JSONB NOT NULL DEFAULT '[]',
    checks_num INTEGER NOT NULL DEFAULT 1,
    failed_to_check boolean NOT NULL DEFAULT false,
    PRIMARY KEY(author, permlink)
  );

  CREATE TABLE posts_votes (
    account ACCOUNT REFERENCES profiles(account) NOT NULL,
    permlink TEXT NOT NULL,
    author ACCOUNT REFERENCES profiles(account) NOT NULL,
    post_unique REAL NOT NULL,
    PRIMARY KEY(account, permlink)
  );

  INSERT INTO posts_plagiarism(author, permlink)
  SELECT author, permlink FROM comments;

-- +migrate Down
  DROP TABLE (posts_plagiarism);
  DROP TABLE (posts_votes);
