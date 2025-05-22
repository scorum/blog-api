-- +migrate Up
CREATE DOMAIN account VARCHAR(16) NOT NULL;

CREATE TABLE profiles (
  account account,
  username TEXT NOT NULL UNIQUE,
  avatar_url TEXT NOT NULL DEFAULT '',
  cover_url TEXT NOT NULL DEFAULT '',
  bio TEXT NOT NULL DEFAULT '',
  location TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (account)
);

CREATE TABLE followers (
  account account REFERENCES profiles(account),
  follow_account account REFERENCES profiles(account),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (account, follow_account)
);

CREATE TABLE media (
  account account REFERENCES profiles(account),
  id VARCHAR(16),
  url TEXT,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (account, id)
);

CREATE INDEX idx_media_url ON media(url);

-- +migrate Down
DROP TABLE followers;
DROP TABLE media;
DROP TABLE profiles;
DROP DOMAIN account;

