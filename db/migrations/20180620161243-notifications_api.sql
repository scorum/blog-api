-- +migrate Up
CREATE TYPE NOTIFICATION_TYPE AS ENUM('started_follow', 'post_voted', 'comment_voted', 'post_flagged', 'comment_flagged',
  'post_replied', 'comment_replied');

CREATE TABLE notifications (
  id UUID PRIMARY KEY,
  account ACCOUNT REFERENCES profiles(account) NOT NULL,
  timestamp TIMESTAMP NOT NULL,
  is_read BOOLEAN DEFAULT FALSE,
  type NOTIFICATION_TYPE NOT NULL,
  meta JSONB NOT NULL
);

CREATE INDEX notifications_account ON notifications(account);

-- +migrate Down
drop table notifications;
drop type NOTIFICATION_TYPE;