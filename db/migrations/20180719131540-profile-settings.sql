
-- +migrate Up

  CREATE TABLE profile_settings (
  account ACCOUNT REFERENCES profiles(account) NOT NULL,
  enable_email_unseen_notifications BOOLEAN NOT NULL DEFAULT TRUE,

  PRIMARY KEY(account)
);

  INSERT INTO profile_settings(account)
	  SELECT account FROM profiles;

-- +migrate Down

DROP TABLE profile_settings;
