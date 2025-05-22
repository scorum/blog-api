-- +migrate Up
CREATE TABLE drafts(
  account ACCOUNT REFERENCES profiles(account),
  id VARCHAR(16) NOT NULL,
  title text,
  body text,
  json_metadata text,

  PRIMARY KEY(account, id)
);

-- +migrate Down

DROP TABLE drafts;
