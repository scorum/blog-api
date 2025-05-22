-- +migrate Up
CREATE TABLE blacklist(
  account TEXT NOT NULL,
  permlink TEXT NOT NULL,
  PRIMARY KEY(account, permlink)
);

-- +migrate Down
DROP TABLE blacklist;
