-- +migrate Up
ALTER TABLE blacklist ADD CONSTRAINT blacklist_account_fk FOREIGN KEY (account) REFERENCES profiles(account);

-- +migrate Down
ALTER TABLE blacklist DROP CONSTRAINT blacklist_account_fk;