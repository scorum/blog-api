
-- +migrate Up
CREATE TABLE push_tokens (
  account ACCOUNT REFERENCES profiles(account) NOT NULL,
  token TEXT NOT NULL,
  PRIMARY KEY (account, token)
);

-- +migrate Down
DROP TABLE push_tokens;