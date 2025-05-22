-- +migrate Up

CREATE TYPE downvote_reason AS ENUM (
    'spam',
    'disagree_on_rewards',
    'plagiarism',
    'hate_or_trolling',
    'low_quality_content'
  );


CREATE TABLE downvotes (
    account ACCOUNT REFERENCES profiles(account) NOT NULL,
    author ACCOUNT REFERENCES profiles(account) NOT NULL,
    permlink TEXT NOT NULL,
    reason downvote_reason NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    PRIMARY KEY(account, author, permlink)
  );

-- +migrate Down
DROP TABLE (downvotes);
DROP TYPE downvote_reason;
