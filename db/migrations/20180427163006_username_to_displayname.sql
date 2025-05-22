-- +migrate Up
ALTER TABLE profiles DROP CONSTRAINT profiles_username_key;
ALTER TABLE profiles RENAME username TO display_name;

-- +migrate Down
ALTER TABLE profiles RENAME display_name TO username;
ALTER TABLE profiles ADD CONSTRAINT profiles_username_key UNIQUE (username);