-- +migrate Up
CREATE TABLE accounts_monitor (
  block_num INT NOT NULL DEFAULT 0
);
INSERT INTO accounts_monitor VALUES(0);

-- +migrate Down
DROP TABLE accounts_monitor;